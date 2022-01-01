module mit-6.824

go 1.16

require mapreduce v0.0.0

require labrpc v0.0.0

require labgob v0.0.0

require raft v0.0.0

require linearizability v0.0.0

replace mapreduce => ./src/mapreduce

replace labrpc => ./src/labrpc

replace labgob => ./src/labgob

replace raft => ./src/raft

// replace kvraft => ./src/kvraft

replace linearizability => ./src/linearizability
