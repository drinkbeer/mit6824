# Notes for Lab2

## Raft Structure

Server State:
* Follower: server is passive, issue no requests on their own but simply respond to requests from Leaders and Candidates 
* Leader: handles all client requests
* Candidate: is used to elect a new leader





## Lab2A

---
`go test -run 2A`

## Lab2B

---
`go test -run 2B` or `time go test -run 2B`

## Lab2C

---
`go test -run 2C`

## Reference
1. [Lab Notes for Spring 2021] https://zhuanlan.zhihu.com/p/359672032
2. [Lab Notes for 2018] https://github.com/double-free/MIT6.824-2018-Chinese
3. [Lab Codes for 2018] https://github.com/wqlin/mit-6.824-2018
4. [Course Notes] https://github.com/aQuaYi/MIT-6.824-Distributed-Systems