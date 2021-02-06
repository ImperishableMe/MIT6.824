package raft

//
// this is an outline of the API that raft must expose to
// the service (or tester). see comments below for
// each of these functions for more details.
//
// rf = Make(...)
//   create a new Raft server.
// rf.Start(command interface{}) (index, term, isleader)
//   start agreement on a new log entry
// rf.GetState() (term, isLeader)
//   ask a Raft for its current term, and whether it thinks it is leader
// ApplyMsg
//   each time a new entry is committed to the log, each Raft peer
//   should send an ApplyMsg to the service (or tester)
//   in the same server.
//

import (
	"math/rand"
	"sync"
	"time"
)
import "sync/atomic"
import "../labrpc"

// import "bytes"
// import "../labgob"

// consts here

const (
	HeartBeatTimeOut = 100 * time.Millisecond
	ElectionTimeOutMin = 400
	ElectionTimeOutMax = 500
)

type State int

const (
	Leader		State = 0
	Candidate	State = 1
	Follower 	State = 2
)
//
// as each Raft peer becomes aware that successive log entries are
// committed, the peer should send an ApplyMsg to the service (or
// tester) on the same server, via the applyCh passed to Make(). set
// CommandValid to true to indicate that the ApplyMsg contains a newly
// committed log entry.
//
// in Lab 3 you'll want to send other kinds of messages (e.g.,
// snapshots) on the applyCh; at that point you can add fields to
// ApplyMsg, but set CommandValid to false for these other uses.
//
type ApplyMsg struct {
	CommandValid bool
	Command      interface{}
	CommandIndex int
}

//
// A Go object implementing a single Raft peer.
//
type Raft struct {
	mu        		sync.Mutex          	// Lock to protect shared access to this peer's state
	peers     		[]*labrpc.ClientEnd 	// RPC end points of all peers
	persister 		*Persister          	// Object to hold this peer's persisted state
	me        		int                 // this peer's index into peers[]
	dead      		int32               // set by Kill()
// persistent
	currentTerm 	int
	votedFor		int 				// which peer got vote from me in currentTerm (votedFor can be me)
	log 			[]Log				// first index is 1

// non volatile
	commitIndex 	int 				// index of highest log entry known to be committed (
									// initialized to 0, increases monotonically)
	lastApplied 	int 				// index of highest log entry applied to state machine (

	state 			State 				// current State of the raft instance
	lastHeartBeat 	time.Time
	electionTimeOut	int
}

type Log struct {
	Cmd string
	Term int
}

// return currentTerm and whether this server
// believes it is the leader.
func (rf *Raft) GetState() (int, bool) {

	var term int
	var isLeader bool

	rf.mu.Lock()
	defer rf.mu.Unlock()

	term = rf.currentTerm
	isLeader = rf.state == Leader
	// Your code here (2A).
	return term, isLeader
}

//
// save Raft's persistent state to stable storage,
// where it can later be retrieved after a crash and restart.
// see paper's Figure 2 for a description of what should be persistent.
//
func (rf *Raft) persist() {
	// Your code here (2C).
	// Example:
	// w := new(bytes.Buffer)
	// e := labgob.NewEncoder(w)
	// e.Encode(rf.xxx)
	// e.Encode(rf.yyy)
	// data := w.Bytes()
	// rf.persister.SaveRaftState(data)
}


//
// restore previously persisted state.
//
func (rf *Raft) readPersist(data []byte) {
	if data == nil || len(data) < 1 { // bootstrap without any state?
		return
	}
	// Your code here (2C).
	// Example:
	// r := bytes.NewBuffer(data)
	// d := labgob.NewDecoder(r)
	// var xxx
	// var yyy
	// if d.Decode(&xxx) != nil ||
	//    d.Decode(&yyy) != nil {
	//   error...
	// } else {
	//   rf.xxx = xxx
	//   rf.yyy = yyy
	// }
}




//
// example RequestVote RPC arguments structure.
// field names must start with capital letters!
//
type RequestVoteArgs struct {
	// Your data here (2A, 2B).
	Term 				int  	// candidate's term
	CandidateId 		int   // candidate requesting vote
	LastLogIndex 		int 	// index of candidate's last log entry
	LastLogTerm 		int 	// term of candidate's last log entry
}

//
// example RequestVote RPC reply structure.
// field names must start with capital letters!
//
type RequestVoteReply struct {
	// Your data here (2A).
	Term				int	// currentTerm, for candidate to update itself
	VoteGranted 		bool 	// true means candidates received vote
}

//
// example RequestVote RPC handler.
//
func (rf *Raft) RequestVote(args *RequestVoteArgs, reply *RequestVoteReply) {
	// Your code here (2A, 2B).
	rf.mu.Lock()
	defer rf.mu.Unlock()

	rf.preRPCHandler(args.Term) // update current term, votedFor

	reply.Term = rf.currentTerm
	reply.VoteGranted = false

	if args.Term < rf.currentTerm {
		return
	}
	if rf.votedFor == -1 || rf.votedFor == args.CandidateId {
		// did not vote for this term or has voted this candidate, repeated req
		isEmpty := len(rf.log) == 0
		if isEmpty {
			reply.VoteGranted = true
			rf.votedFor = args.CandidateId
			rf.resettingElectionTimer()
			return
		}
		// not empty
		myLastLog := rf.log[len(rf.log) - 1]

		if myLastLog.Term < args.LastLogTerm ||
			myLastLog.Term == args.LastLogTerm &&
			len(rf.log) <= args.LastLogIndex {

			reply.VoteGranted = true
			rf.votedFor = args.CandidateId
			rf.resettingElectionTimer()
			return
		}
	}

}

//
// example code to send a RequestVote RPC to a server.
// server is the index of the target server in rf.peers[].
// expects RPC arguments in args.
// fills in *reply with RPC reply, so caller should
// pass &reply.
// the types of the args and reply passed to Call() must be
// the same as the types of the arguments declared in the
// handler function (including whether they are pointers).
//
// The labrpc package simulates a lossy network, in which servers
// may be unreachable, and in which requests and replies may be lost.
// Call() sends a request and waits for a reply. If a reply arrives
// within a timeout interval, Call() returns true; otherwise
// Call() returns false. Thus Call() may not return for a while.
// A false return can be caused by a dead server, a live server that
// can't be reached, a lost request, or a lost reply.
//
// Call() is guaranteed to return (perhaps after a delay) *except* if the
// handler function on the server side does not return.  Thus there
// is no need to implement your own timeouts around Call().
//
// look at the comments in ../labrpc/labrpc.go for more details.
//
// if you're having trouble getting RPC to work, check that you've
// capitalized all field names in structs passed over RPC, and
// that the caller passes the address of the reply struct with &, not
// the struct itself.
//
func (rf *Raft) sendRequestVote(server int, args *RequestVoteArgs, reply *RequestVoteReply) bool {
	ok := rf.peers[server].Call("Raft.RequestVote", args, reply)
	return ok
}


type AppendEntriesArgs struct {
	Term			int 		//  leader's term
	LeaderId 		int 		//	for follower's to redirect client
	PrevLogIndex	int
	PrevLogTerm 	int
	Entries			[]Log
	LeaderCommit 	int
}


type AppendEntriesReply struct {
	Term 			int 		// currentTerm, for leader to update itself
	Success			bool 		// true, if follower contained entry matching
								// PrevLogIndex and PrevLogTerm
}


func (rf *Raft) AppendEntries(args *AppendEntriesArgs, reply *AppendEntriesReply) {
	// Your code here (2A, 2B).
	rf.mu.Lock()
	defer rf.mu.Unlock()

	newRPC := rf.preRPCHandler(args.Term)

	reply.Term = rf.currentTerm
	reply.Success = true

	if !newRPC {
		reply.Term = rf.currentTerm
		reply.Success = false
	} else {
		rf.resettingElectionTimer()
	}

}


func (rf *Raft) sendAppendEntries(server int, args *AppendEntriesArgs, reply *AppendEntriesReply) bool {
	ok := rf.peers[server].Call("Raft.AppendEntries", args, reply)
	return ok
}
//
// the service using Raft (e.g. a k/v server) wants to start
// agreement on the next command to be appended to Raft's log. if this
// server isn't the leader, returns false. otherwise start the
// agreement and return immediately. there is no guarantee that this
// command will ever be committed to the Raft log, since the leader
// may fail or lose an election. even if the Raft instance has been killed,
// this function should return gracefully.
//
// the first return value is the index that the command will appear at
// if it's ever committed. the second return value is the current
// term. the third return value is true if this server believes it is
// the leader.
//
func (rf *Raft) Start(command interface{}) (int, int, bool) {
	index := -1
	term := -1
	isLeader := true

	// Your code here (2B).


	return index, term, isLeader
}

//
// the tester doesn't halt goroutines created by Raft after each test,
// but it does call the Kill() method. your code can use killed() to
// check whether Kill() has been called. the use of atomic avoids the
// need for a lock.
//
// the issue is that long-running goroutines use memory and may chew
// up CPU time, perhaps causing later tests to fail and generating
// confusing debug output. any goroutine with a long-running loop
// should call killed() to check whether it should stop.
//
func (rf *Raft) Kill() {
	atomic.StoreInt32(&rf.dead, 1)
	// Your code here, if desired.
}

func (rf *Raft) killed() bool {
	z := atomic.LoadInt32(&rf.dead)
	return z == 1
}

//
// the service or tester wants to create a Raft server. the ports
// of all the Raft servers (including this one) are in peers[]. this
// server's port is peers[me]. all the servers' peers[] arrays
// have the same order. persister is a place for this server to
// save its persistent state, and also initially holds the most
// recent saved state, if any. applyCh is a channel on which the
// tester or service expects Raft to send ApplyMsg messages.
// Make() must return quickly, so it should start goroutines
// for any long-running work.
//
func Make(peers []*labrpc.ClientEnd, me int,
	persister *Persister, applyCh chan ApplyMsg) *Raft {
	rf := &Raft{}
	rf.peers = peers
	rf.persister = persister
	rf.me = me
	rf.state = Follower
	rf.votedFor = -1
	rf.resettingElectionTimer()
	// Your initialization code here (2A, 2B, 2C).

	// initialize from state persisted before a crash
	rf.readPersist(persister.ReadRaftState())

	DPrintf("[%d] is On!", rf.me)

	go rf.electionDaemon()
	go rf.appendEntriesDaemon()

	return rf
}

func (rf *Raft) electionDaemon() {

	for !rf.killed() {
		time.Sleep(30 * time.Millisecond)

		rf.mu.Lock()

		if time.Since(rf.lastHeartBeat).Milliseconds() >
			int64(rf.electionTimeOut) {

			if rf.state != Leader {
				go rf.kickOffAnElection()
			}
		}
		rf.mu.Unlock()
	}
}

func (rf *Raft) kickOffAnElection() {
	rf.mu.Lock()

	voteCount := 1  // voting for itself
	result := 1
	rf.votedFor = rf.me
	rf.state = Candidate
	rf.currentTerm++
	rf.resettingElectionTimer()

	DPrintf("[%d] Starting An election at term #%d", rf.me, rf.currentTerm)
	askedVoteTerm := rf.currentTerm // for local use

	rf.mu.Unlock()

	for ind,_ := range rf.peers {
		if ind == rf.me {
			continue
		}
		go func(curInd int){
			args := RequestVoteArgs{
				Term:         askedVoteTerm,
				CandidateId:  rf.me,
				LastLogIndex: 0,
				LastLogTerm:  0,
			}
			reply := RequestVoteReply{}
			ok := rf.sendRequestVote(curInd, &args, &reply)
			if !ok {		// error in the RPC, so no vote :3
				return
			}

			rf.mu.Lock()
			defer rf.mu.Unlock()
			result++

			rf.preRPCHandler(reply.Term) // check whether you had an old term

			if rf.currentTerm != askedVoteTerm || rf.state != Candidate {
				return
			}
			if reply.VoteGranted {
				voteCount++  // use synchronous variable here
			}
		}(ind)
	}

	for {
		rf.mu.Lock()
		if voteCount * 2 < len(rf.peers) && result < len(rf.peers) {
			rf.mu.Unlock()
			time.Sleep(10 * time.Millisecond)
		} else {
			break
		}
	}
	defer rf.mu.Unlock()

	if rf.state != Candidate || rf.currentTerm != askedVoteTerm {
		return
	}
	if voteCount * 2 >= len(rf.peers) {
		// won the selection
		rf.becomeLeader()
		DPrintf("[%d] is becoming the leader at term #%d", rf.me, rf.currentTerm)
		go rf.sendHeartBeat()
		//
	} else {
		// become follower
		rf.becomeFollower()
		DPrintf("[%d] lost the election at term #%d", rf.me, rf.currentTerm)
	}
}

func (rf *Raft) appendEntriesDaemon() {

	for !rf.killed() {
		// kick off append entries
		time.Sleep(HeartBeatTimeOut)
		rf.mu.Lock()
		if rf.state == Leader {
			go rf.sendHeartBeat()
		}
		rf.mu.Unlock()
	}
}

func (rf *Raft) sendHeartBeat() {
	rf.mu.Lock()
	defer rf.mu.Unlock()

	for ind, _ := range rf.peers {
		if ind == rf.me {
			continue
		}
		go rf.sendHeartBeatToOne(ind, rf.currentTerm)
	}
}

func (rf *Raft) sendHeartBeatToOne(server, term int)  {

	for !rf.killed() {
		rf.mu.Lock()
		args := AppendEntriesArgs{
			Term:         term,
			LeaderId:     rf.me,
			PrevLogIndex: 0,
			PrevLogTerm:  0,
			Entries:      nil,
			LeaderCommit: 0,
		}
		reply := AppendEntriesReply{}
		rf.mu.Unlock()

		ok := rf.sendAppendEntries(server, &args, &reply)
		if !ok {
			continue // appendEntries failed, try again!
		}
		rf.mu.Lock()
		if term != rf.currentTerm || rf.state != Leader {
			rf.mu.Unlock()
			break
		}
		if reply.Term > rf.currentTerm {
			rf.currentTerm = reply.Term
			rf.becomeFollower()
			rf.mu.Unlock()
			break
		}
		if reply.Success {
			rf.mu.Unlock()
			break
		}
		rf.mu.Unlock()
	}
}

// will be called holding the lock
func (rf *Raft) resettingElectionTimer() {
	rf.lastHeartBeat = time.Now()
	rf.electionTimeOut = rand.Intn(ElectionTimeOutMax - ElectionTimeOutMin) +
		ElectionTimeOutMin
}

// It will always be called holding mu lock
func (rf *Raft) preRPCHandler(foundTerm int) bool {
	if foundTerm > rf.currentTerm {
		rf.currentTerm = foundTerm
		rf.becomeFollower()
		return false
	}
	return true
}
// Called after holding mu lock

func (rf *Raft) becomeLeader() {


	rf.state = Leader
	rf.resettingElectionTimer()
	// need some nextInd resetting for 2B, 2C
}

// Also called holding the mu lock
func (rf *Raft) becomeFollower(){
	rf.votedFor = -1
	rf.state = Follower
}
