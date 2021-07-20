package raft

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
