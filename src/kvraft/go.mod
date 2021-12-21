module kvraft

go 1.17

require raft v0.0.0
require labrpc v0.0.0
require labgob v0.0.0
require linearizability v0.0.0

replace raft => ../raft
replace labrpc => ../labrpc
replace labgob => ../labgob
replace linearizability => ../linearizability
