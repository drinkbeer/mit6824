package raft

//
// this is an outline of the API that raft must expose to
// the service (or tester). see comments below for
// each of these functions for more details.
//
// rf = Make(...)
//   create a new Raft server.
// rf.Start(command interface{}) (index, term, isleader)
//   start agreement on a new log entry
// rf.GetState() (term, isLeader)
//   ask a Raft for its current term, and whether it thinks it is leader
// ApplyMsg
//   each time a new entry is committed to the log, each Raft peer
//   should send an ApplyMsg to the service (or tester)
//   in the same server.
//

import (
	"fmt"
	"raft/common"
	"sync"
	"sync/atomic"
	"time"

	"labrpc"
)

const (
	HeartbeatInterval    = time.Duration(120) * time.Millisecond
	ElectionTimeoutLower = time.Duration(300) * time.Millisecond
	ElectionTimeoutUpper = time.Duration(400) * time.Millisecond
)

//
// ApplyMsg as each Raft peer becomes aware that successive log entries are
// committed, the peer should send an ApplyMsg to the service (or
// tester) on the same server, via the applyCh passed to Make(). set
// CommandValid to true to indicate that the ApplyMsg contains a newly
// committed log entry.
//
// in Lab 3 you'll want to send other kinds of messages (e.g.,
// snapshots) on the applyCh; at that point you can add fields to
// ApplyMsg, but set CommandValid to false for these other uses.
//
type ApplyMsg struct {
	CommandValid bool
	Command      interface{}
	CommandIndex int
}

//
// Raft is a Go object implementing a single Raft peer.
//
type Raft struct {
	mu        sync.Mutex          // Lock to protect shared access to this peer's state
	peers     []*labrpc.ClientEnd // RPC end points of all peers
	persister *Persister          // Object to hold this peer's persisted state
	me        int                 // this peer's index into peers[]

	// Your data here (2A, 2B, 2C).
	// Look at the paper's Figure 2 for a description of what
	// state a Raft server must maintain.

	// 2A
	// from Figure 2

	// Persistent state on all servers:
	currentTerm int
	votedFor    int
	electionTimer  *time.Timer
	heartbeatTimer *time.Timer
	state          NodeState
}

// NodeState indicates the state of current node.
type NodeState string

const (
	Follower  = "follower"
	Candidate = "candidate"
	Leader    = "leader"
)

func (rf *Raft) convertTo(s NodeState) {
	if s == rf.state {
		return
	}

	rf.state = s
	switch s {
	case Follower:
		rf.heartbeatTimer.Stop()
		common.ResetTimer(rf.electionTimer, common.RandomDuration(ElectionTimeoutLower, ElectionTimeoutUpper))
		rf.votedFor = -1
	case Candidate:
		rf.startElection()
	case Leader:
		rf.electionTimer.Stop()
		//rf.broadcastHeartbeat()
		common.ResetTimer(rf.heartbeatTimer, HeartbeatInterval)
	}
	common.Debug("Term %d: me %d convert from %v to %v \n", rf.currentTerm, rf.me, rf.state, s)
}

// GetState return currentTerm and whether this server
// believes it is the leader.
func (rf *Raft) GetState() (int, bool) {
	rf.mu.Lock()
	defer rf.mu.Unlock()

	var term int
	var isleader bool

	// Your code here (2A).
	term = rf.currentTerm
	isleader = rf.state == Leader

	return term, isleader
}

// persist save Raft's persistent state to stable storage,
// where it can later be retrieved after a crash and restart.
// see paper's Figure 2 for a description of what should be persistent.
func (rf *Raft) persist() {
	// Your code here (2C).
	// Example:
	// w := new(bytes.Buffer)
	// e := labgob.NewEncoder(w)
	// e.Encode(rf.xxx)
	// e.Encode(rf.yyy)
	// data := w.Bytes()
	// rf.persister.SaveRaftState(data)
}

// readPersist restore previously persisted state.
func (rf *Raft) readPersist(data []byte) {
	if data == nil || len(data) < 1 { // bootstrap without any state?
		return
	}
	// Your code here (2C).
	// Example:
	// r := bytes.NewBuffer(data)
	// d := labgob.NewDecoder(r)
	// var xxx
	// var yyy
	// if d.Decode(&xxx) != nil ||
	//    d.Decode(&yyy) != nil {
	//   error...
	// } else {
	//   rf.xxx = xxx
	//   rf.yyy = yyy
	// }
}

// RequestVoteArgs example RequestVote RPC arguments structure.
// field names must start with capital letters!
type RequestVoteArgs struct {
	// Your data here (2A, 2B).

	// 2A
	Term        int
	CandidateId int
}

//
// RequestVoteReply example RequestVote RPC reply structure.
// field names must start with capital letters!
//
type RequestVoteReply struct {
	// Your data here (2A).

	// 2A
	Term        int
	VoteGranted bool
}

// RequestVote RPC handler.
func (rf *Raft) RequestVote(args *RequestVoteArgs, reply *RequestVoteReply) {
	// Your code here (2A, 2B).

	rf.mu.Lock()
	defer rf.mu.Unlock()

	// 2A
	if args.Term < rf.currentTerm || (args.Term == rf.currentTerm && rf.votedFor != -1 && rf.votedFor != args.CandidateId) {
		//fmt.Printf("RequestVote failed - me: %v, currentTerm: %v, argsTerm: %v, votedFor: %v, argsCandidatedId: %v \n", rf.me, rf.currentTerm, args.Term, rf.votedFor, args.CandidateId)
		reply.Term = rf.currentTerm
		reply.VoteGranted = false
		return
	}

	if args.Term > rf.currentTerm {
		//fmt.Printf("RequestVote - me %v, term %v is not leader, another Candidate %v, term %v has higher term than me \n", rf.me, rf.currentTerm, args.CandidateId, args.Term)
		rf.currentTerm = args.Term
		rf.convertTo(Follower)
	}

	// 2B

	// 2A
	// vote for the candidate in the RequestVoteArgs
	rf.votedFor = args.CandidateId
	reply.Term = rf.currentTerm // rely currentTerm so the peer who RequestVote knows their term is lower or higher than me
	reply.VoteGranted = true
	common.ResetTimer(rf.electionTimer, common.RandomDuration(ElectionTimeoutLower, ElectionTimeoutUpper))
	fmt.Printf("RequestVote - me %v term %v voted for candidate %v term %v \n", rf.me, rf.currentTerm, args.CandidateId, args.Term)
}

type AppendEntriesArgs struct {
	// 2A
	Term     int
	LeaderId int
}

type AppendEntriesReply struct {
	// 2A
	Term    int
	Success int
}

//func (rf *Raft) AppendEntries(args *AppendEntriesArgs, reply *AppendEntriesReply) {
//
//}

func (rf Raft) broadcastHeartbeat() {

}

func (rf *Raft) startElection() {
	// 2A
	rf.currentTerm += 1

	args := RequestVoteArgs{
		Term:        rf.currentTerm,
		CandidateId: rf.me,
	}

	var voteCount int32

	for peer := range rf.peers {
		if peer == rf.me {
			rf.votedFor = rf.me
			atomic.AddInt32(&voteCount, 1)
			continue
		}
		go func(server int) {
			var reply RequestVoteReply
			if rf.sendRequestVote(server, &args, &reply) {
				rf.mu.Lock()
				if reply.VoteGranted && rf.state == Candidate {
					fmt.Printf("sendRequestVote succeed, reply.VoteGranted %v, me %v state %v, state == candidate: %v \n", reply.VoteGranted, rf.me, rf.state, rf.state == Candidate)
					atomic.AddInt32(&voteCount, 1)
					if atomic.LoadInt32(&voteCount) > int32(len(rf.peers)/2) {
						rf.convertTo(Leader)
						fmt.Printf("%v got elected to Leader \n", rf.me)
					}
				} else {
					if reply.Term > rf.currentTerm {
						fmt.Printf("%v doesn't get enough votes, still Candidate \n", rf.me)
						rf.currentTerm = reply.Term
						rf.convertTo(Follower)
					}
					fmt.Printf("startElection - reply.VoteGranted %v, me %v state %v \n", reply.VoteGranted, rf.me, rf.state)
				}
				rf.mu.Unlock()
			}
		}(peer)
	}
}

//
// sendRequestVote example code to send a RequestVote RPC to a server.
// server is the index of the target server in rf.peers[].
// expects RPC arguments in args.
// fills in *reply with RPC reply, so caller should
// pass &reply.
// the types of the args and reply passed to Call() must be
// the same as the types of the arguments declared in the
// handler function (including whether they are pointers).
//
// The labrpc package simulates a lossy network, in which servers
// may be unreachable, and in which requests and replies may be lost.
// Call() sends a request and waits for a reply. If a reply arrives
// within a timeout interval, Call() returns true; otherwise
// Call() returns false. Thus Call() may not return for a while.
// A false return can be caused by a dead server, a live server that
// can't be reached, a lost request, or a lost reply.
//
// Call() is guaranteed to return (perhaps after a delay) *except* if the
// handler function on the server side does not return.  Thus there
// is no need to implement your own timeouts around Call().
//
// look at the comments in ../labrpc/labrpc.go for more details.
//
// if you're having trouble getting RPC to work, check that you've
// capitalized all field names in structs passed over RPC, and
// that the caller passes the address of the reply struct with &, not
// the struct itself.
//
func (rf *Raft) sendRequestVote(server int, args *RequestVoteArgs, reply *RequestVoteReply) bool {
	callResult := rf.peers[server].Call("Raft.RequestVote", args, reply)
	//if callResult {
	//	fmt.Printf("sendRequestVote succeed \n")
	//}
	return callResult
}

//
// Start the service using Raft (e.g. a k/v server) wants to start
// agreement on the next command to be appended to Raft's log. if this
// server isn't the leader, returns false. otherwise start the
// agreement and return immediately. there is no guarantee that this
// command will ever be committed to the Raft log, since the leader
// may fail or lose an election. even if the Raft instance has been killed,
// this function should return gracefully.
//
// the first return value is the index that the command will appear at
// if it's ever committed. the second return value is the current
// term. the third return value is true if this server believes it is
// the leader.
//
func (rf *Raft) Start(command interface{}) (int, int, bool) {
	index := -1
	term := -1
	isLeader := true

	// Your code here (2B).

	return index, term, isLeader
}

//
// Kill the tester calls Kill() when a Raft instance won't
// be needed again. you are not required to do anything
// in Kill(), but it might be convenient to (for example)
// turn off debug output from this instance.
//
func (rf *Raft) Kill() {
	// Your code here, if desired.
}

//
// Make the service or tester wants to create a Raft server. the ports
// of all the Raft servers (including this one) are in peers[]. this
// server's port is peers[me]. all the servers' peers[] arrays
// have the same order. persister is a place for this server to
// save its persistent state, and also initially holds the most
// recent saved state, if any. applyCh is a channel on which the
// tester or service expects Raft to send ApplyMsg messages.
// Make() must return quickly, so it should start goroutines
// for any long-running work.
//
func Make(peers []*labrpc.ClientEnd, me int, persister *Persister, applyCh chan ApplyMsg) *Raft {
	rf := &Raft{}
	rf.peers = peers
	rf.persister = persister
	rf.me = me

	// Your initialization code here (2A, 2B, 2C).
	rf.currentTerm = 0
	rf.votedFor = -1
	rf.heartbeatTimer = time.NewTimer(HeartbeatInterval)
	rf.electionTimer = time.NewTimer(common.RandomDuration(ElectionTimeoutLower, ElectionTimeoutUpper))
	rf.state = Follower

	// initialize from state persisted before a crash
	rf.mu.Lock()
	rf.readPersist(persister.ReadRaftState())
	rf.mu.Unlock()

	go func(node *Raft) {
		for {
			select {
			case <-node.electionTimer.C:
				node.mu.Lock()
				// electionTimer has elapsed, no need to call Stop()
				node.electionTimer.Reset(common.RandomDuration(ElectionTimeoutLower, ElectionTimeoutUpper))
				if node.state == Follower {
					//fmt.Printf("ElectionTimer elapsed: Follower -> Candidate \n")
					node.convertTo(Candidate)
				} else {
					//fmt.Printf("ElectionTimer elapsed: Candidate/Leader starts election \n")
					//rf.mu.Lock()
					node.startElection()
					//rf.mu.Unlock()
				}
				node.mu.Unlock()

			case <-node.heartbeatTimer.C:
				node.mu.Lock()
				if node.state == Leader {
					fmt.Printf("HeartbeatTimer elapsed: Leader broadcasts Heartbeat \n")
					node.broadcastHeartbeat()
					node.heartbeatTimer.Reset(HeartbeatInterval)
				}
				node.mu.Unlock()
			}
		}
	}(rf)

	return rf
}
