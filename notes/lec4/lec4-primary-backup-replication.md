# Notes for Lec4

## COURSE STRUCTURE

---
#### Structure
1. Primary/Backup Replication for Fault Tolerance
2. Case study of VMware FT, an extreme version of the idea

A bit more about Fault Tolerance:
1. to provide availability
2. despite server and network failures
3. using replication


## MAIN TOPICS

---
#### What kinds of failures can replication deal with?
1. `fail-stop` failure of a single replica
   1. fan stops working, CPU overheats and shuts itself down
   2. human interruption, e.g. someone trips over replica's power cord or network cable
   3. software notices it is out of disk space and stops
2. Maybe not defects in h/w or bugs in s/w or human configuration errors
   1. Often not `fail-stop`
   2. Maybe correlated (i.e. cause all replicas to crash at the same time)
   3. But, sometimes can be detected (e.g. checksums)
3. How about earthquake or city-wide power failure?
   1. Only if replicas are physically separated

#### Is replication worth the Nx expense?

#### Two main replication approaches:
1. State transfer
   1. Primary replica executes the service
   2. Primary sends [new] state to backups
2. Replicated state machine
   1. Clients send operations to primary,
      1. primary sequences and sends to backups
   2. All replicas execute all operations
   3. If all primary/replicas have same start state,
      1. same operations
      2. same order
      3. deterministic
      4. then same end state

State transfer is simpler, but state may be large, slow to transfer over network.

Replicated state machine often generates less network traffic, because operations are often small compared to state. But complex to get right. VM-FT uses replicated state machine, as do Labs 2/3/4.

#### Big Questions to think about?
- What state to replicate?
- Does primary have to wait for backup?
- When to cut over to backup?
- Are anomalies visible at cut-over?
- How to bring a replacement backup up to speed?

#### At what level do we want replicas to be identical?
- Application state, e.g. a database's tables?
  - GFS works this way
  - Can be efficient; primary only sends high-level operations to backup
  - Application code (server) must understand fault tolerance, to e.g. forward op stream
- Machine level, e.g. registers and RAM content?
  - might allow us to replicate any existing server w/o modification!
  - requires forwarding of machine events (interrupts, DMA, &c)
  - requires "machine" modifications to send/recv event stream...

#### Today's paper (VMware FT) replicates machine-level state
- Transparent: can run any existing O/S and server software!
- Appears like a single server to clients


## CASE STUDY: VMware FT

---
#### Overview
[diagram: app, O/S, VM-FT underneath, disk server, network, clients]  
- Terminology/Words:
  - hypervisor == monitor == VMM (virtual machine monitor)
  - O/S+app is the "guest" running inside a virtual machine
- Two machines, primary and backup
- Primary sends all external events (client packets &c) to backup over network "logging channel", carrying log entries
- Ordinarily, backup's output is suppressed by FT
- If either stops being able to talk to the other over the network "goes live" and provides sole service
  - If primary goes live, it stops sending log entries to the backup


## Reference
1. Notes: http://nil.csail.mit.edu/6.824/2020/notes/l-vm-ft.txt