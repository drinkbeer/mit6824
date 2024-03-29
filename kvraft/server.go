package raftkv

import (
	"fmt"
	"mit6824/labgob"
	"mit6824/labrpc"
	"mit6824/raft"
	"sync"
	"time"
)

type Op struct {
	// Your definitions here.
	// Field names must start with capital letters,
	// otherwise RPC will break.
	// 3A
	Key   string
	Value string
	Name  string

	ClientId  int64
	RequestId int
}

type Notification struct {
	ClientId  int64
	RequestId int
}

type KVServer struct {
	mu      sync.Mutex
	me      int
	rf      *raft.Raft
	applyCh chan raft.ApplyMsg

	maxraftstate int // snapshot if log grows this big

	// Your definitions here.
	// 3A
	db                   map[string]string
	dispatcher           map[int]chan Notification
	lastAppliedRequestId map[int64]int

	// 3B

}

func (kv *KVServer) waitApplying(op Op, timeout time.Duration) bool {
	index, _, isLeader := kv.rf.Start(op)
	if !isLeader {
		return true
	}

	var wrongLeader bool
	defer DPrintf("kvserver %d got %s() RPC, insert op %++v at %d, reply WrongLeader = %v",
		kv.me, op.Name, op, index, wrongLeader)

	kv.mu.Lock()
	if _, ok := kv.dispatcher[index]; !ok {
		kv.dispatcher[index] = make(chan Notification, 1)
	}
	ch := kv.dispatcher[index]
	kv.mu.Unlock()

	select {
	case notify := <-ch:
		kv.mu.Lock()
		delete(kv.dispatcher, index)
		kv.mu.Unlock()
		if notify.ClientId != op.ClientId || notify.RequestId != op.RequestId {
			wrongLeader = true
		} else {
			wrongLeader = false
		}
	case <-time.After(timeout):
		wrongLeader = true
	}
	return wrongLeader
}

func (kv *KVServer) Get(args *GetArgs, reply *GetReply) {
	// Your code here.
	// 3A
	op := Op{
		Key:       args.Key,
		Name:      "Get",
		ClientId:  args.ClientId,
		RequestId: args.RequestId,
	}

	fmt.Printf("Get op: %#v \n", op)

	reply.WrongLeader = kv.waitApplying(op, 500*time.Millisecond)

	if !reply.WrongLeader {
		kv.mu.Lock()
		value, ok := kv.db[args.Key]
		kv.mu.Unlock()
		if ok {
			reply.Value = value
			return
		}

		// not found
		reply.Err = ErrNoKey
	}
}

func (kv *KVServer) PutAppend(args *PutAppendArgs, reply *PutAppendReply) {
	// Your code here.
	// 3A
	op := Op{
		Key:       args.Key,
		Value:     args.Value,
		Name:      args.Op,
		ClientId:  args.ClientId,
		RequestId: args.RequestId,
	}

	fmt.Printf("PutAppend op: %#v \n", op)
	reply.WrongLeader = kv.waitApplying(op, 500*time.Millisecond)
}

//
// the tester calls Kill() when a KVServer instance won't
// be needed again. you are not required to do anything
// in Kill(), but it might be convenient to (for example)
// turn off debug output from this instance.
//
func (kv *KVServer) Kill() {
	kv.rf.Kill()
	// Your code here, if desired.
}

//
// servers[] contains the ports of the set of
// servers that will cooperate via Raft to
// form the fault-tolerant key/value service.
// me is the index of the current server in servers[].
// the k/v server should store snapshots through the underlying Raft
// implementation, which should call persister.SaveStateAndSnapshot() to
// atomically save the Raft state along with the snapshot.
// the k/v server should snapshot when Raft's saved state exceeds maxraftstate bytes,
// in order to allow Raft to garbage-collect its log. if maxraftstate is -1,
// you don't need to snapshot.
// StartKVServer() must return quickly, so it should start goroutines
// for any long-running work.
//
func StartKVServer(servers []*labrpc.ClientEnd, me int, persister *raft.Persister, maxraftstate int) *KVServer {
	// call labgob.Register on structures you want
	// Go's RPC library to marshall/unmarshall.
	labgob.Register(Op{})

	kv := new(KVServer)
	kv.me = me
	kv.maxraftstate = maxraftstate

	// You may need initialization code here.
	kv.db = make(map[string]string)
	kv.dispatcher = make(map[int]chan Notification)
	kv.lastAppliedRequestId = make(map[int64]int)

	kv.applyCh = make(chan raft.ApplyMsg)
	kv.rf = raft.Make(servers, me, persister, kv.applyCh)

	// You may need initialization code here.
	go func() {
		for msg := range kv.applyCh {
			if !msg.CommandValid {
				continue
			}

			op := msg.Command.(Op)
			DPrintf("kvserver %d applied command %s at index %d", kv.me, op.Name, msg.CommandIndex)
			kv.mu.Lock()
			lastAppliedRequestId, ok := kv.lastAppliedRequestId[op.ClientId]
			if !ok || lastAppliedRequestId < op.RequestId {
				switch op.Name {
				case "Put":
					kv.db[op.Key] = op.Value
				case "Append":
					kv.db[op.Key] += op.Value
				case "Get":
				}
				kv.lastAppliedRequestId[op.ClientId] = op.RequestId
			}
			ch, ok := kv.dispatcher[msg.CommandIndex]
			kv.mu.Unlock()

			if ok {
				notify := Notification{
					ClientId:  op.ClientId,
					RequestId: op.RequestId,
				}
				ch <- notify
			}
		}
	}()

	return kv
}
