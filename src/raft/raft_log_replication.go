package raft

type LogEntry struct {
	Command interface{}
	Term    int
}

type AppendEntriesArgs struct {
	Term     int // 2A
	LeaderId int // 2A

	PrevLogIndex int        // 2B
	PrevLogTerm  int        // 2B
	LogEntries   []LogEntry // 2B
	LeaderCommit int        // 2B
}

type AppendEntriesReply struct {
	Term    int  // 2A
	Success bool // 2A

	// OPTIMIZE: see thesis section 5.3
	ConflictTerm  int // 2C
	ConflictIndex int // 2C
}

func (rf *Raft) AppendEntries(args *AppendEntriesArgs, reply *AppendEntriesReply) {
	rf.mu.Lock()
	defer rf.mu.Unlock()
	defer rf.persist() // execute before rf.mu.Unlock()
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

	//if args.PrevLogIndex <= rf.snapshottedIndex {
	//	reply.Success = true
	//
	//	// sync log if needed
	//	if args.PrevLogIndex+len(args.LogEntries) > rf.snapshottedIndex {
	//		// if snapshottedIndex == prevLogIndex, all log entries should be added.
	//		startIdx := rf.snapshottedIndex - args.PrevLogIndex
	//		// only keep the last snapshotted one
	//		rf.logs = rf.logs[:1]
	//		rf.logs = append(rf.logs, args.LogEntries[startIdx:]...)
	//	}
	//
	//	return
	//}

	// entries before args.PrevLogIndex might be unmatch
	// return false and ask Leader to decrement PrevLogIndex
	absoluteLastLogIndex := rf.getAbsoluteLogIndex(len(rf.logs) - 1)
	if absoluteLastLogIndex < args.PrevLogIndex {
		reply.Success = false
		reply.Term = rf.currentTerm
		// optimistically thinks receiver's log matches with Leader's as a subset
		reply.ConflictIndex = absoluteLastLogIndex + 1
		// no conflict term
		reply.ConflictTerm = -1
		return
	}

	if rf.logs[rf.getRelativeLogIndex(args.PrevLogIndex)].Term != args.PrevLogTerm {
		reply.Success = false
		reply.Term = rf.currentTerm
		// receiver's log in certain term unmatches Leader's log
		reply.ConflictTerm = rf.logs[rf.getRelativeLogIndex(args.PrevLogIndex)].Term

		// expecting Leader to check the former term
		// so set ConflictIndex to the first one of entries in ConflictTerm
		//conflictIndex := args.PrevLogIndex
		//// apparently, since rf.logs[0] are ensured to match among all servers
		//// ConflictIndex must be > 0, safe to minus 1
		//for rf.logs[rf.getRelativeLogIndex(conflictIndex-1)].Term == reply.ConflictTerm {
		//	conflictIndex--
		//	if conflictIndex == rf.snapshottedIndex+1 {
		//		// this may happen after snapshot,
		//		// because the term of the first log may be the current term
		//		// before lab 3b this is not going to happen, since rf.logs[0].Term = 0
		//		break
		//	}
		//}
		//reply.ConflictIndex = conflictIndex
		return
	}

	// compare from rf.logs[args.PrevLogIndex + 1]
	unmatch_idx := -1
	for idx := range args.LogEntries {
		if len(rf.logs) < rf.getRelativeLogIndex(args.PrevLogIndex+2+idx) ||
			rf.logs[rf.getRelativeLogIndex(args.PrevLogIndex+1+idx)].Term != args.LogEntries[idx].Term {
			// unmatch log found
			unmatch_idx = idx
			break
		}
	}

	if unmatch_idx != -1 {
		// there are unmatch entries
		// truncate unmatch Follower entries, and apply Leader entries
		rf.logs = rf.logs[:rf.getRelativeLogIndex(args.PrevLogIndex+1+unmatch_idx)]
		rf.logs = append(rf.logs, args.LogEntries[unmatch_idx:]...)
	}

	// Leader guarantee to have all committed entries
	// TODO: Is that possible for lastLogIndex < args.LeaderCommit?
	if args.LeaderCommit > rf.commitIndex {
		absoluteLastLogIndex := rf.getAbsoluteLogIndex(len(rf.logs) - 1)
		if args.LeaderCommit <= absoluteLastLogIndex {
			rf.setCommitIndex(args.LeaderCommit)
		} else {
			rf.setCommitIndex(absoluteLastLogIndex)
		}
	}

	reply.Success = true
}

// should be called with lock
func (rf *Raft) broadcastHeartbeat() {
	for i := range rf.peers {
		if i == rf.me {
			continue
		}
		go func(server int) {
			rf.mu.Lock()
			if rf.state != Leader {
				rf.mu.Unlock()
				return
			}

			prevLogIndex := rf.nextIndex[server] - 1

			//if prevLogIndex < rf.snapshottedIndex {
			//	// leader has discarded log entries the follower needs
			//	// send snapshot to follower and retry later
			//	rf.mu.Unlock()
			//	rf.syncSnapshotWith(server)
			//	return
			//}
			// use deep copy to avoid race condition
			// when override log in AppendEntries()
			entries := make([]LogEntry, len(rf.logs[rf.getRelativeLogIndex(prevLogIndex+1):]))
			copy(entries, rf.logs[rf.getRelativeLogIndex(prevLogIndex+1):])

			args := AppendEntriesArgs{
				Term:         rf.currentTerm,
				LeaderId:     rf.me,
				PrevLogIndex: prevLogIndex,
				PrevLogTerm:  rf.logs[rf.getRelativeLogIndex(prevLogIndex)].Term,
				LogEntries:   entries,
				LeaderCommit: rf.commitIndex,
			}
			rf.mu.Unlock()

			var reply AppendEntriesReply
			if rf.sendAppendEntries(server, &args, &reply) {
				rf.mu.Lock()
				if rf.state != Leader {
					rf.mu.Unlock()
					return
				}
				if reply.Success {
					// successfully replicated args.LogEntries
					rf.matchIndex[server] = args.PrevLogIndex + len(args.LogEntries)
					rf.nextIndex[server] = rf.matchIndex[server] + 1

					// check if we need to update commitIndex
					// from the last log entry to committed one
					for i := rf.getAbsoluteLogIndex(len(rf.logs) - 1); i > rf.commitIndex; i-- {
						count := 0
						for _, matchIndex := range rf.matchIndex {
							if matchIndex >= i {
								count += 1
							}
						}

						if count > len(rf.peers)/2 {
							// most of nodes agreed on rf.logs[i]
							rf.setCommitIndex(i)
							break
						}
					}

				} else {
					if reply.Term > rf.currentTerm {
						rf.currentTerm = reply.Term
						rf.convertTo(Follower)
						rf.persist()
					}
					//else {
					//	// log unmatch, update nextIndex[server] for the next trial
					//	rf.nextIndex[server] = reply.ConflictIndex
					//
					//	// if term found, override it to
					//	// the first entry after entries in ConflictTerm
					//	if reply.ConflictTerm != -1 {
					//		DPrintf("%v conflict with server %d, prevLogIndex %d, log length = %d",
					//			rf, server, args.PrevLogIndex, len(rf.logs))
					//		for i := args.PrevLogIndex; i >= rf.snapshottedIndex+1; i-- {
					//			if rf.logs[rf.getRelativeLogIndex(i-1)].Term == reply.ConflictTerm {
					//				// in next trial, check if log entries in ConflictTerm matches
					//				rf.nextIndex[server] = i
					//				break
					//			}
					//		}
					//	}
					//
					//	// TODO: retry now or in next RPC?
					//}
				}
				rf.mu.Unlock()
			}
		}(i)
	}
}

func (rf *Raft) sendAppendEntries(server int, args *AppendEntriesArgs, reply *AppendEntriesReply) bool {
	ok := rf.peers[server].Call("Raft.AppendEntries", args, reply)
	return ok
}

func (rf *Raft) getRelativeLogIndex(index int) int {
	// index of rf.logs
	return index
}

func (rf *Raft) getAbsoluteLogIndex(index int) int {
	// index of log including snapshotted ones
	return index
}

func (rf *Raft) setCommitIndex(commitIndex int) {
	rf.commitIndex = commitIndex
	// apply all entries between lastApplied and committed
	// should be called after commitIndex updated
	if rf.commitIndex > rf.lastApplied {
		Debug("%v apply from index %d to %d", rf, rf.lastApplied+1, rf.commitIndex)
		entriesToApply := append([]LogEntry{},
			rf.logs[rf.getRelativeLogIndex(rf.lastApplied+1):rf.getRelativeLogIndex(rf.commitIndex+1)]...)

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
				if rf.lastApplied < msg.CommandIndex {
					rf.lastApplied = msg.CommandIndex
				}
				rf.mu.Unlock()
			}
		}(rf.lastApplied+1, entriesToApply)
	}
}
