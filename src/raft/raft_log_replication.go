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
	PrevLogTerm  int        // Term of prevLogIndex entry, PrevLog is the first log in Entries, is PrevLogTerm redundant?
	Entries      []LogEntry // Log entries to store (empty for heartbeat; may send more than one for efficiency)
	LeaderCommit int        // Leader's commitIndex
}

type AppendEntriesReply struct {
	// 2B
	Term    int  // Follower or candidate's term, for leader to update itself
	Success bool // True if follower or candidate contained entry matching PrevLogIndex and PrevLogTerm

	// OPTIMIZE: see thesis section 5.3
	ConflictTerm  int // 2C
	ConflictIndex int // 2C
}

// AppendEntries RPC handler is used (1) by leader to replicate log entries; (2) used as heartbeat
func (rf *Raft) AppendEntries(args *AppendEntriesArgs, reply *AppendEntriesReply) {
	rf.mu.Lock()
	defer rf.mu.Unlock()
	defer rf.persist() // 2C execute before rf.mu.Unlock()
	if args.Term < rf.currentTerm {
		reply.Success = false
		reply.Term = rf.currentTerm
		return
	}

	if args.Term > rf.currentTerm {
		rf.currentTerm = args.Term
		rf.convertTo(Follower)
		// do not return here.
	}

	// reset election timer even log does not match
	// args.LeaderId is the current term's Leader
	ResetTimer(rf.electionTimer,
		RandomDuration(ElectionTimeoutLower, ElectionTimeoutUpper))

	// entries before args.PrevLogIndex might be unmatch
	// return false and ask Leader to decrement PrevLogIndex
	/*
		- - - - - -
		          ^
				  |
				  absoluteLastLogIndex = 5
	*/
	absoluteLastLogIndex := len(rf.logs) - 1
	if absoluteLastLogIndex < args.PrevLogIndex {
		reply.Success = false
		reply.Term = rf.currentTerm
		// optimistically thinks receiver's log matches with Leader's as a subset
		reply.ConflictIndex = absoluteLastLogIndex + 1
		// no conflict term
		reply.ConflictTerm = -1
		return
	}

	if rf.logs[args.PrevLogIndex].Term != args.PrevLogTerm {
		reply.Success = false
		reply.Term = rf.currentTerm
		// receiver's log in certain term unmatches Leader's log
		reply.ConflictTerm = rf.logs[args.PrevLogIndex].Term

		// expecting Leader to check the former term
		// so set ConflictIndex to the first one of entries in ConflictTerm
		conflictIndex := args.PrevLogIndex
		// apparently, since rf.logs[0] are ensured to match among all servers
		// ConflictIndex must be > 0, safe to minus 1
		for rf.logs[conflictIndex-1].Term == reply.ConflictTerm {
			conflictIndex--
		}
		reply.ConflictIndex = conflictIndex
		return
	}

	// compare from rf.logs[args.PrevLogIndex + 1]
	/*
		Ideally:
			- - - - - -
			          ^
					  |
					  len(rf.logs)-1

			- - - - - - - - -
					^ ^
					| |
					args.PrevLogIndex
					  |
					  entry starts

		Sometimes:
		    - - - - - -
			          ^
					  |
					  len(rf.logs)-1

			- - - - - - - - -
				^ ^     ^
				| |     |
				args.PrevLogIndex
				  |     |
				  entry starts
				        |
						unmatch_idx (copy should start here)
	*/
	unmatch_idx := -1
	for idx := range args.Entries {
		if len(rf.logs)-1 < args.PrevLogIndex+1+idx ||
			rf.logs[args.PrevLogIndex+1+idx].Term != args.Entries[idx].Term {
			// unmatch log found
			unmatch_idx = idx
			break
		}
	}

	if unmatch_idx != -1 {
		// there are unmatch entries
		// truncate unmatch Follower entries, and apply Leader entries
		rf.logs = rf.logs[:args.PrevLogIndex+1+unmatch_idx]
		rf.logs = append(rf.logs, args.Entries[unmatch_idx:]...)
	}

	// Leader guarantee to have all committed entries
	// TODO: Is that possible for lastLogIndex < args.LeaderCommit?
	if args.LeaderCommit > rf.commitIndex {
		lastLogIndex := len(rf.logs) - 1
		if args.LeaderCommit <= lastLogIndex {
			rf.setCommitIndex(args.LeaderCommit)
		} else {
			rf.setCommitIndex(lastLogIndex)
		}
	}

	reply.Success = true
}

func (rf *Raft) broadcastHeartbeat() {
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

			entries := make([]LogEntry, len(rf.logs[prevLogIndex+1:]))
			copy(entries, rf.logs[prevLogIndex+1:]) // log (index = prevLogIndex) is the entry that matched, so we make this matched entry as first one in the entries sent to follower, ensure at least one entry matched

			args := AppendEntriesArgs{
				Term:         rf.currentTerm,
				LeaderId:     rf.me,
				PrevLogIndex: prevLogIndex,
				PrevLogTerm:  rf.logs[prevLogIndex].Term,
				Entries:      entries,
				LeaderCommit: rf.commitIndex,
			}
			rf.mu.Unlock()

			var reply AppendEntriesReply
			if rf.sendAppendEntries(server, &args, &reply) {
				rf.mu.Lock()
				// if rf.state != Leader {
				// 	rf.mu.Unlock()
				// 	return
				// }
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
							rf.setCommitIndex(i)
							break
						}
					}
				} else {
					if reply.Term > rf.currentTerm {
						rf.currentTerm = reply.Term
						rf.convertTo(Follower)
						rf.persist()
					} else {
						// log unmatch, update nextIndex[server] for the next trial
						rf.nextIndex[server] = reply.ConflictIndex

						// if term found, override it to
						// the first entry after entries in ConflictTerm
						if reply.ConflictTerm != -1 {
							Debug("%v conflict with server %d, prevLogIndex %d, log length = %d",
								rf, server, args.PrevLogIndex, len(rf.logs))
							for i := args.PrevLogIndex; i >= 1; i-- {
								if rf.logs[i-1].Term == reply.ConflictTerm {
									// in next trial, check if log entries in ConflictTerm matches
									rf.nextIndex[server] = i
									break
								}
							}
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

func (rf *Raft) setCommitIndex(commitIndex int) {
	rf.commitIndex = commitIndex

	if rf.commitIndex > rf.lastApplied {
		Debug("%v apply from index %d to %d \n", rf.me, rf.lastApplied+1, rf.commitIndex)
		entriesToApply := append([]LogEntry{},
			rf.logs[rf.lastApplied+1:rf.commitIndex+1]...)

		go func(startIdx int, entries []LogEntry) {
			for idx, entry := range entries {
				var msg ApplyMsg
				msg.CommandValid = true
				msg.Command = entry.Command
				msg.CommandIndex = startIdx + idx
				rf.applyCh <- msg
				// do not forget to update lastApplied index
				// this is another goroutine, so protect it with lock
				rf.mu.Lock()
				rf.lastApplied = msg.CommandIndex
				rf.mu.Unlock()
			}
		}(rf.lastApplied+1, entriesToApply)
	}
}
