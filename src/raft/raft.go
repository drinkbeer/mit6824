package raft

import (
	"labrpc"
	"sync"
	"time"
)

/*
this is an outline of the API that raft must expose to
the service (or tester). see comments below for
each of these functions for more details.

rf = Make(...)
  create a new Raft server.
rf.Start(command interface{}) (index, term, isleader)
  start agreement on a new log entry
rf.GetState() (term, isLeader)
  ask a Raft for its current term, and whether it thinks it is leader
ApplyMsg
  each time a new entry is committed to the log, each Raft peer
  should send an ApplyMsg to the service (or tester)
  in the same server.
*/

const (
	HeartbeatInterval    = time.Duration(120) * time.Millisecond
	ElectionTimeoutLower = time.Duration(300) * time.Millisecond
	ElectionTimeoutUpper = time.Duration(400) * time.Millisecond
)

// ApplyMsg as each Raft peer becomes aware that successive log entries are
// committed, the peer should send an ApplyMsg to the service (or
// tester) on the same server, via the applyCh passed to Make(). set
// CommandValid to true to indicate that the ApplyMsg contains a newly
// committed log entry.
//
// in Lab 3 you'll want to send other kinds of messages (e.g.,
// snapshots) on the applyCh; at that point you can add fields to
// ApplyMsg, but set CommandValid to false for these other uses.
type ApplyMsg struct {
	CommandValid bool
	Command      interface{}
	CommandIndex int
}

// Raft is a Go object implementing a single Raft peer.
type Raft struct {
	mu        sync.Mutex          // Lock to protect shared access to this peer's state
	peers     []*labrpc.ClientEnd // RPC end points of all peers
	persister *Persister          // Object to hold this peer's persisted state
	me        int                 // This peer's index into peers[]

	// Your data here (2A, 2B, 2C).
	// Look at the paper's Figure 2 for a description of what
	// state a Raft server must maintain.

	// 2A, from Figure 2
	currentTerm    int // Latest term server has seen
	votedFor       int // CandidateId that received vote in current term
	electionTimer  *time.Timer
	heartbeatTimer *time.Timer
	state          NodeState // State of server
	// 2B, from Figure 2
	logs        []LogEntry // Each entry contains command for state machine, and term when entry was received by leader (term starts from 1)
	applyCh     chan ApplyMsg
	commitIndex int   // Index of highest log entry known to be committed
	lastApplied int   // Index of highest log entry applied to state machine
	nextIndex   []int // For each server, index of the next log entry to send to that server
	matchIndex  []int // For each server, index of highest log entry known to be replicated on server
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
func (rf *Raft) Start(command interface{}) (int, int, bool) {
	index := -1
	term := -1
	isLeader := true

	// Your code here (2B).
	term, isLeader = rf.GetState()
	if isLeader {
		rf.mu.Lock()
		rf.logs = append(rf.logs, LogEntry{Command: command, Term: term})
		index = len(rf.logs) - 1 // index that the command appears at in leader's log
		rf.matchIndex[rf.me] = index
		rf.nextIndex[rf.me] = index + 1
		// start agreement now
		rf.broadcastHeartbeat()
		rf.mu.Unlock()
	}

	return index, term, isLeader
}

// Kill the tester calls Kill() when a Raft instance won't
// be needed again. you are not required to do anything
// in Kill(), but it might be convenient to (for example)
// turn off debug output from this instance.
func (rf *Raft) Kill() {
	// Your code here, if desired.
}

// Make the service or tester wants to create a Raft server. the ports
// of all the Raft servers (including this one) are in peers[]. this
// server's port is peers[me]. all the servers' peers[] arrays
// have the same order. persister is a place for this server to
// save its persistent state, and also initially holds the most
// recent saved state, if any. applyCh is a channel on which the
// tester or service expects Raft to send ApplyMsg messages.
// Make() must return quickly, so it should start goroutines
// for any long-running work.
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
	rf.logs = make([]LogEntry, 1)
	rf.applyCh = applyCh
	rf.nextIndex = make([]int, len(rf.peers))
	for i := range rf.nextIndex {
		// initialized to leader last log index + 1
		rf.nextIndex[i] = len(rf.logs)
	}
	rf.matchIndex = make([]int, len(rf.peers))
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
					ResetTimer(node.heartbeatTimer, HeartbeatInterval)
				}
				node.mu.Unlock()
			}
		}
	}(rf)

	return rf
}
