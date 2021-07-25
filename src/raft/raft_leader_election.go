package raft

import "sync/atomic"

// convertTo converts the current Raft node to different state.
func (rf *Raft) convertTo(s NodeState) {
	if s == rf.state {
		return
	}

	rf.state = s
	switch s {
	case Follower:
		rf.heartbeatTimer.Stop()
		ResetTimer(rf.electionTimer, RandomDuration(ElectionTimeoutLower, ElectionTimeoutUpper))
		rf.votedFor = -1
	case Candidate:
		rf.startElection()
	case Leader:
		rf.electionTimer.Stop()
		rf.broadcastHeartbeat()
		ResetTimer(rf.heartbeatTimer, HeartbeatInterval)
	}
	Debug("Term %d: me %d convert from %v to %v \n", rf.currentTerm, rf.me, rf.state, s)
}

// RequestVoteArgs example RequestVote RPC arguments structure.
// field names must start with capital letters!
// Your data here (2A, 2B).
type RequestVoteArgs struct {
	// 2A
	Term        int // Candidate's term
	CandidateId int // Candidate's Id (who is requesting vote)

	// 2B
	LastLogIndex int // Index of candidate's last log entry
	LastLogTerm  int // Term of candidate's last log entry
}

// RequestVoteReply example RequestVote RPC reply structure.
// field names must start with capital letters!
// Your data here (2A).
type RequestVoteReply struct {
	// 2A
	Term        int  // Candidate's current term
	VoteGranted bool // true means candidate received vote
}

// RequestVote RPC handler.
// Your code here (2A, 2B).
func (rf *Raft) RequestVote(args *RequestVoteArgs, reply *RequestVoteReply) {
	// 2A
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

	// 2B: candidate's vote should be at least up-to-date as receiver's log
	// "up-to-date" is defined in thesis 5.4.1
	lastLogIndex := len(rf.logs) - 1
	if args.LastLogTerm < rf.logs[lastLogIndex].Term || (args.LastLogTerm == rf.logs[lastLogIndex].Term && args.LastLogIndex < lastLogIndex) {
		// Receiver is more up-to-date, does not grant vote
		reply.Term = rf.currentTerm
		reply.VoteGranted = false
		return
	}

	// 2A
	// vote to the candidate in the RequestVoteArgs
	rf.votedFor = args.CandidateId
	reply.Term = rf.currentTerm // rely currentTerm so the peer who RequestVote knows their term is lower or higher than me
	reply.VoteGranted = true
	ResetTimer(rf.electionTimer, RandomDuration(ElectionTimeoutLower, ElectionTimeoutUpper))
	Debug("RequestVote - me %v term %v voted for candidate %v term %v \n", rf.me, rf.currentTerm, args.CandidateId, args.Term)
}

// startElection starts to send RequestVote RPC to peers, and count the votes on conversion to candidate.
func (rf *Raft) startElection() {
	// 2A
	rf.currentTerm += 1

	args := RequestVoteArgs{
		Term:        rf.currentTerm,
		CandidateId: rf.me,
	}

	// startElection is called on conversion to candidate, and election timer is reset before calling startElection.
	//ResetTimer(rf.electionTimer, RandomDuration(ElectionTimeoutLower, ElectionTimeoutUpper))

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
func (rf *Raft) sendRequestVote(server int, args *RequestVoteArgs, reply *RequestVoteReply) bool {
	callResult := rf.peers[server].Call("Raft.RequestVote", args, reply)
	return callResult
}
