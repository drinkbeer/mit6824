## Notes for Lec1

## COURSE STRUCTURE

---
#### What is distributed system?
* multiple cooperating computers
* storage for big web sites, MapReduce, peer-to-peer sharing, etc
* lots of critical infrastructure is distributed

#### Why need distributed system and the backwards?
* to increase capacity via parallelism
* to increase faults via replication
* to place computing physically close to external entities
* to achieve security via isolation

But:
* many concurrent parts, complex interactions
* must cope with partial failures
* tricky to realize performance potential

#### Components of learning?
1. lectures
2. papers
3. two exams
4. labs
5. final projects (optional)

Labs:
1. MapReduce
2. replication for fault-tolerance using Raft
3. fault-tolerant key/value store
4. shared key/value store

## MAIN TOPICS

---
The couse is about infrastructure for applications.
* Storage
* Communication
* Computation

The big goal: abstractions that hide the complexity of distribution. A couple of topics that will come repeatly in the course.

#### Topic: implementation
* RPC, threads, concurrency control.
* The labs...

#### Topic: performance

* The goal: scalable throughput
    - Nx servers -> Nx total throughput via parallel CPU, disk, net.
    - So handling more load only requires buying more computers. Rather than re-design by expensive programmers.
    - Effective when you can divide work w/o much interaction.
* Scaling gets harder as N grows:
    - Load im-balance, stragglers, slowest-of-N latency.
    - Non-parallelizable code: initialization, interaction.
    - Bottlenecks from shared resources, e.g. network.
* Some performance problems aren't easily solved by scaling
    - e.g. quick response time for a single user request
    - e.g. all users want to update the same data often requires better design rather than just more computers
* Lab 4

#### Topic: fault tolerance
* 1000s of servers, big network -> always something broken
* We'd like to hide these failures from the application.
* We often want: 
    - Availability - app can make progress despite failures
    - Recoverability - app will come back to life when failures are repaired
* Big idea: replicated servers.
    - If one server crashes, can proceed using the other(s).
    - Labs 1, 2 and 3

#### Topic: consistency
* General-purpose infrastructure needs well-defined behavior. E.g. "Get(k) yields the value from the most recent Put(k, v)."
* Achieving good behavior is hard!
    - "Replica" servers are hard to keep identical.
    - Clients may crash midway through multi-step update.
    - Servers may crash, e.g. after executing but before replying.
    - Network partition may take live servers look dead; risk of "split brain".
* Consistency and performance are enermies.
    - Strong consistency requires communication,
        - e.g. Get() much check for a recent Put().
    - Many designs provide only weak consistency, to gain speed.
        - e.g. Get() does *not* yield the latest Put()!
        Painful for application programmers but may be a good trade-off.
* Many design points are possible in the consistency/performance spectrum!

## CASE STUDY: MapReduce

---
#### MapReduce overview
* Context: multi-hour computations on multi-terabyte data-sets
    - e.g. build search index, or sort, or analyze structure of web
    - only practical with 1000s of computers
    - applications not written by distributed systems experts
* Overall goal: easy for non-specialists programmers
* Programmers just defines Map and Reduce functions, often fairly simple sequential code.
* MR takes care of, and hides, all aspects of distribution!

#### Abstract view of a MapReduce job
* input is (already) split into M files
```
  Input1 -> Map -> a,1 b,1
  Input2 -> Map ->     b,1
  Input3 -> Map -> a,1     c,1
                    |   |   |
                    |   |   -> Reduce -> c,1
                    |   -----> Reduce -> b,2
                    ---------> Reduce -> a,2
```
* MR calls Map() for each input file, produces set of k2,v2
    "intermediate" data
    each Map() call is a "task"
* MR gathers all intermediate v2's for a given k2,
    and passes each key + values to a Reduce call
* final output is set of <k2,v3> pairs from Reduce()s

Example: word count
* input is thousands of text files
* Map(k, v)
    * split v into words
    * for each word w
        * emit (w, "1")
* Reduce(k, v)
    * emit(len(v))



### Reference
1. [Notes of Lec1 from course](https://pdos.csail.mit.edu/6.824/notes/l01.txt)