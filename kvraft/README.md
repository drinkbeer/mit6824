# Notes for Lab3

## Lab3A


---
`go test . -run 3A`

```
➜  mit6824 git:(master) ✗ go test ./kvraft -run 3A
^C%
➜  mit6824 git:(master) ✗ go test ./kvraft -run checkClntAppends
ok      mit6824/kvraft  0.148s [no tests to run]
➜  mit6824 git:(master) ✗ go test ./kvraft -run checkConcurrentAppends
ok      mit6824/kvraft  0.167s [no tests to run]
➜  mit6824 git:(master) ✗ go test ./kvraft -run partitioner
ok      mit6824/kvraft  0.142s [no tests to run]
➜  mit6824 git:(master) ✗ go test ./kvraft -run GenericTest
ok      mit6824/kvraft  0.155s [no tests to run]
➜  mit6824 git:(master) ✗ go test ./kvraft -run GenericTestLinearizability
ok      mit6824/kvraft  0.152s [no tests to run]
➜  mit6824 git:(master) ✗ go test ./kvraft -run TestBasic3A
ok      mit6824/kvraft  15.743s
➜  mit6824 git:(master) ✗ go test ./kvraft -run TestConcurrent3A
ok      mit6824/kvraft  17.034s
➜  mit6824 git:(master) ✗ go test ./kvraft -run TestUnreliable3A
ok      mit6824/kvraft  17.800s
➜  mit6824 git:(master) ✗ go test ./kvraft -run TestUnreliableOneKey3A
ok      mit6824/kvraft  3.027s
➜  mit6824 git:(master) ✗ go test ./kvraft -run TestOnePartition3A
ok      mit6824/kvraft  3.451s
➜  mit6824 git:(master) ✗ go test ./kvraft -run TestManyPartitionsOneClient3A
ok      mit6824/kvraft  23.112s
➜  mit6824 git:(master) ✗ go test ./kvraft -run TestManyPartitionsManyClients3A
ok      mit6824/kvraft  24.182s
➜  mit6824 git:(master) ✗ go test ./kvraft -run TestPersistOneClient3A
ok      mit6824/kvraft  19.841s
➜  mit6824 git:(master) ✗ go test ./kvraft -run TestPersistConcurrent3A
ok      mit6824/kvraft  21.307s
➜  mit6824 git:(master) ✗ go test ./kvraft -run TestPersistConcurrentUnreliable3A
ok      mit6824/kvraft  21.946s
➜  mit6824 git:(master) ✗ go test ./kvraft -run TestPersistPartition3A
ok      mit6824/kvraft  28.286s
➜  mit6824 git:(master) ✗ go test ./kvraft -run TestPersistPartitionUnreliable3A
ok      mit6824/kvraft  29.973s
➜  mit6824 git:(master) ✗ go test ./kvraft -run TestPersistPartitionUnreliableLinearizable3A
ok      mit6824/kvraft  26.124s
```

## Lab3B

---
`go test . -run 3B`

## Reference
1. [Lab3 Docs](http://nil.csail.mit.edu/6.824/2018/labs/lab-kvraft.html)
