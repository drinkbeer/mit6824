# Notes for Lab2

## Raft Structure

Server State:
* Follower: server is passive, issue no requests on their own but simply respond to requests from Leaders and Candidates 
* Leader: handles all client requests
* Candidate: is used to elect a new leader





## Lab2A

---
#### Make

Makes a Raft object. Starts a goroutine processing the `electionTimer` and `heartbeatTimer`.
```
func Make(peers []*labrpc.ClientEnd, me int, persister *Persister, applyCh chan ApplyMsg) *Raft {
	rf := &Raft{}
	rf.peers = peers
	rf.persister = persister
	rf.me = me

	// Your initialization code here (2A, 2B, 2C).
	// 2A
	rf.currentTerm = 0
	rf.votedFor = -1
	rf.heartbeatTimer = time.NewTimer(HeartbeatInterval)
	rf.electionTimer = time.NewTimer(RandomDuration(ElectionTimeoutLower, ElectionTimeoutUpper))
	rf.state = Follower
	// 2B
	// 2C

	// initialize from state persisted before a crash
	// 2C
	rf.mu.Lock()
	rf.readPersist(persister.ReadRaftState())
	rf.mu.Unlock()

	go func(node *Raft) {
		for {
			select {
			case <-node.electionTimer.C:
				node.mu.Lock()
				// electionTimer has elapsed, no need to call Stop()
				ResetTimer(node.electionTimer, RandomDuration(ElectionTimeoutLower, ElectionTimeoutUpper))
				if node.state == Follower {
					node.convertTo(Candidate)
				} else {
					node.startElection()
				}
				node.mu.Unlock()

			case <-node.heartbeatTimer.C:
				node.mu.Lock()
				if node.state == Leader {
					Debug("HeartbeatTimer elapsed: Leader broadcasts Heartbeat \n")
					node.broadcastHeartbeat()
					node.heartbeatTimer.Reset(HeartbeatInterval)
				}
				node.mu.Unlock()
			}
		}
	}(rf)

	return rf
}
```
#### startElection
```
// startElection starts to send RequestVote RPC to peers, and count the votes.
func (rf *Raft) startElection() {
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
					Debug("sendRequestVote succeed, reply.VoteGranted %v, me %v state %v, state == candidate: %v \n", reply.VoteGranted, rf.me, rf.state, rf.state == Candidate)
					atomic.AddInt32(&voteCount, 1)
					if atomic.LoadInt32(&voteCount) > int32(len(rf.peers)/2) {
						rf.convertTo(Leader)
						Debug("%v got elected to Leader \n", rf.me)
					}
				} else {
					if reply.Term > rf.currentTerm {
						rf.currentTerm = reply.Term
						rf.convertTo(Follower)
					}
				}
				rf.mu.Unlock()
			}
		}(peer)
	}
}
```

#### RequestVote RPC
```
func (rf *Raft) RequestVote(args *RequestVoteArgs, reply *RequestVoteReply) {
	rf.mu.Lock()
	defer rf.mu.Unlock()
	if args.Term < rf.currentTerm || (args.Term == rf.currentTerm && rf.votedFor != -1 && rf.votedFor != args.CandidateId) {
		reply.Term = rf.currentTerm
		reply.VoteGranted = false
		return
	}

	if args.Term > rf.currentTerm {
		rf.currentTerm = args.Term
		rf.convertTo(Follower)
	}

	// vote to the candidate in the RequestVoteArgs
	rf.votedFor = args.CandidateId
	reply.Term = rf.currentTerm // rely currentTerm so the peer who RequestVote knows their term is lower or higher than me
	reply.VoteGranted = true
	ResetTimer(rf.electionTimer, RandomDuration(ElectionTimeoutLower, ElectionTimeoutUpper))
	Debug("RequestVote - me %v term %v voted for candidate %v term %v \n", rf.me, rf.currentTerm, args.CandidateId, args.Term)
}
```

#### Run Test
`go test . -run 2A` or `go test ./raft -run 2A`

## Lab2B

---
`go test . -run 2B` or `time go test . -run 2B` or `go test ./raft -run 2B`

## Lab2C

```
func (rf *Raft) persist() {
	w := new(bytes.Buffer)
	e := labgob.NewEncoder(w)
	e.Encode(rf.currentTerm)
	e.Encode(rf.votedFor)
	e.Encode(rf.logs)
	data := w.Bytes()
	rf.persister.SaveRaftState(data)
}

// readPersist restore previously persisted state.
func (rf *Raft) readPersist(data []byte) {
	if data == nil || len(data) < 1 { // bootstrap without any state?
		return
	}
	r := bytes.NewBuffer(data)
	d := labgob.NewDecoder(r)
	var currentTerm, votedFor int
	var logs []LogEntry
	if d.Decode(&currentTerm) != nil ||
		d.Decode(&votedFor) != nil ||
		d.Decode(&logs) != nil {
		Debug("%v fails to recover from persist", rf)
		return
	}

	rf.currentTerm = currentTerm
	rf.votedFor = votedFor
	rf.logs = logs
}
```

---
`go test . -run 2C` or `go test ./raft -run 2C`

## Reference
1. [Lab2 Docs] http://nil.csail.mit.edu/6.824/2018/labs/lab-raft.html
2. [Lab Notes for 2018] https://github.com/double-free/MIT6.824-2018-Chinese
3. [Lab Codes for 2018] https://github.com/wqlin/mit-6.824-2018
4. [Course Notes] https://github.com/aQuaYi/MIT-6.824-Distributed-Systems
5. [Lab Notes for Spring 2021] https://zhuanlan.zhihu.com/p/359672032
