package raft

type LogEntry struct {
	Command interface{}
	Term    int
}

type AppendEntriesArgs struct {
	// 2B
	Term         int        // Leader's term
	LeaderId     int        // Leader's id, so follower can redirect requests to leader
	PrevLogIndex int        // Index of log entry immediately preceding new ones
	PrevLogTerm  int        // Term of prevLogIndex entry
	Entries      []LogEntry // Log entries to store (empty for heartbeat; may send more than one for efficiency)
	LeaderCommit int        // Leader's commitIndex
}

type AppendEntriesReply struct {
	// 2B
	Term    int  // Follower or candidate's term, for leader to update itself
	Success bool // True if follower or candidate contained entry matching PrevLogIndex and PrevLogTerm
}

// AppendEntries RPC handler is used (1) by leader to replicate log entries; (2) used as heartbeat
func (rf *Raft) AppendEntries(args *AppendEntriesArgs, reply *AppendEntriesReply) {
	rf.mu.Lock()
	defer rf.mu.Unlock()
	defer rf.persist()

	if args.Term < rf.currentTerm {
		reply.Term = rf.currentTerm
		reply.Success = false
		return
	}

	if args.Term > rf.currentTerm {
		rf.currentTerm = args.Term
		if rf.state != Follower {
			// Server's state should be either follower or candidate, as AppendEntries RPC is sent by leader to follower/candidate
			rf.convertTo(Follower)
		}
	}

	ResetTimer(rf.electionTimer, RandomDuration(ElectionTimeoutLower, ElectionTimeoutUpper))

	// lastLogIndex before PrevLogIndex indicates this server's logs are missing in accident,
	// because leader thinks PrevLogIndex is the log index that is matched. If leader and
	// and follower is in sync, lastLogIndex should be equals to PrevLogIndex. If the leader
	// is newly elected leader, the follower could has some extra uncommitted logs, so in
	// this case, lastLogIndex > PrevLogIndex (in the following steps, we will overwrite the
	// uncommitted logs).
	// Return false and ask leader to decrease PrevLogIndex, and then retry
	lastLogIndex := len(rf.logs) - 1
	if lastLogIndex < args.PrevLogIndex {
		reply.Term = rf.currentTerm
		reply.Success = false
		return
	}

	// The term of log in lastLogIndex is different from PrevLogTerm indicates there is
	// conflicts between local logs and leader's logs, return false and ask leader to
	// decrease PrevLogTerm, and then retry
	if rf.logs[args.PrevLogIndex].Term != args.PrevLogTerm {
		reply.Term = rf.currentTerm
		reply.Success = false
		return
	}

	// By now, with the above two 'if', we can confirm that lastLogIndex >= PrevLogIndex, and the PrevLog is matched.

	// Go array slice is half-open rance which includes first element, but excludes the last one
	// So here we set the logs to [0, PrevLogIndex), then concatenate logs with Entities [0, end]
	rf.logs = rf.logs[:args.PrevLogIndex]
	rf.logs = append(rf.logs, args.Entries...)

	if args.LeaderCommit > rf.commitIndex {
		lastLogIndex = len(rf.logs) - 1 // after appending the logs, needs update the lastLogIndex
		if args.LeaderCommit <= lastLogIndex {
			rf.commitIndex = args.LeaderCommit
		} else {
			rf.commitIndex = lastLogIndex
		}
	}

	reply.Success = true
}

func (rf Raft) broadcastHeartbeat() {
	for peer := range rf.peers {
		if peer == rf.me {
			continue
		}

		go func(server int) {
			rf.mu.Lock()
			if rf.state != Leader {
				rf.mu.Unlock()
				return
			}

			prevLogIndex := rf.nextIndex[server] - 1 // the last log index in the follower's logs that matches with leader's

			entries := make([]LogEntry, len(rf.logs[prevLogIndex:]))
			copy(entries, rf.logs[prevLogIndex:]) // log (index = prevLogIndex) is the entry that matched, so we make this matched entry as first one in the entries sent to follower, ensure at least one entry matched

			args := AppendEntriesArgs{
				Term:         rf.currentTerm,
				LeaderId:     rf.me,
				PrevLogIndex: prevLogIndex,
				PrevLogTerm:  rf.logs[prevLogIndex].Term,
				Entries:      entries,
				LeaderCommit: rf.commitIndex,
			}

			var reply AppendEntriesReply
			if rf.sendAppendEntries(server, &args, &reply) {
				rf.mu.Lock()
				if rf.state == Leader {
					rf.mu.Unlock()
					return
				}
				if reply.Success {
					rf.matchIndex[server] = args.PrevLogIndex + len(args.Entries)
					rf.nextIndex[server] = rf.matchIndex[server] + 1

					for i := len(rf.logs) - 1; i > rf.commitIndex; i-- {
						count := 0
						for _, matchedIndex := range rf.matchIndex {
							if matchedIndex >= i {
								count += 1
							}
						}

						if count > len(rf.peers)/2 {
							rf.commitIndex = i
							break
						}
					}
				} else {
					if reply.Term > rf.currentTerm {
						rf.currentTerm = reply.Term
						rf.convertTo(Follower)
					} else {
						// leader and follower's term in PrevLogIndex is not match, decrease PrevLogIndex by one for next retry
						// Retry in next RPC call (Optimization: AppendEntries RPC call right now)
						if rf.nextIndex[server] > 1 {
							rf.nextIndex[server] -= 1
						}
					}
				}
				rf.mu.Unlock()
			}
		}(peer)
	}
}

func (rf *Raft) sendAppendEntries(server int, args *AppendEntriesArgs, reply *AppendEntriesReply) bool {
	ok := rf.peers[server].Call("Raft.AppendEntries", args, reply)
	return ok
}
