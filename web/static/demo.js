// demo.js — Interactive scenario animations for the Distributed Ticket System demo page

// ═══════════════════════════════════════════════════════════════════════
// SCENARIO DEFINITIONS
// Each scenario has:
//   meta:   { icon, title, desc }
//   init:   initial state for each node (state, label, clock)
//   steps:  array of step objects
//
// Step object:
//   title:   short headline
//   detail:  monospace detail line
//   logs:    [ { node:'n1'|'n2'|'n3'|'sys', msg, type:'success'|'error'|'warn'|'info'|'' } ]
//   nodeUpdates: [ { id:1|2|3, state:'leader'|'active'|'down'|'election'|'cs'|'highlight', label, clock } ]
//   arrows:  [ { from:1|2|3, to:1|2|3, label, color:'green'|'blue'|'orange'|'red'|'purple'|'yellow' } ]
// ═══════════════════════════════════════════════════════════════════════

const SCENARIOS = {

    // ──────────────────────────────────────────────────────────────────
    // 1. BULLY ELECTION
    // ──────────────────────────────────────────────────────────────────
    bully: {
        meta: {
            icon: '🗳️',
            title: 'Bully Leader Election',
            desc: 'Node 1 (leader) crashes → nodes detect timeout → Bully algorithm elects highest-ID node as leader'
        },
        init: [
            { id: 1, state: 'leader', label: 'Leader', icon: '👑', clock: 5 },
            { id: 2, state: 'active', label: 'Follower', icon: '🖥️', clock: 3 },
            { id: 3, state: 'active', label: 'Follower', icon: '🖥️', clock: 2 }
        ],
        steps: [{
                title: 'Node 1 (Leader) crashes',
                detail: 'Node 1 stops sending heartbeats — process terminated',
                logs: [
                    { node: 'n1', msg: 'FATAL: process killed — heartbeat stopped', type: 'error' },
                    { node: 'sys', msg: 'Node 1 is now DOWN', type: 'error' }
                ],
                nodeUpdates: [{ id: 1, state: 'down', label: 'DOWN', icon: '💀' }],
                arrows: []
            },
            {
                title: 'Nodes 2 & 3 detect heartbeat timeout',
                detail: 'timeout_ms=5000 exceeded — no heartbeat from Node 1',
                logs: [
                    { node: 'n2', msg: '[heartbeat] timeout — Node 1 unreachable (5002ms)', type: 'warn' },
                    { node: 'n3', msg: '[heartbeat] timeout — Node 1 unreachable (5007ms)', type: 'warn' }
                ],
                nodeUpdates: [
                    { id: 2, state: 'election', label: 'Detecting…' },
                    { id: 3, state: 'election', label: 'Detecting…' }
                ],
                arrows: []
            },
            {
                title: 'Node 3 starts Bully Election',
                detail: 'Node 3 sends ELECTION msg to all higher-ID nodes (Node 2)',
                logs: [
                    { node: 'n3', msg: '[bully] starting election — sending ELECTION to Node 2', type: 'info' },
                    { node: 'n2', msg: '[bully] received ELECTION from Node 3', type: '' }
                ],
                nodeUpdates: [{ id: 3, state: 'election', label: 'Electing' }],
                arrows: [{ from: 3, to: 2, label: 'ELECTION', color: 'orange' }]
            },
            {
                title: 'Node 2 responds OK to Node 3',
                detail: '"I am alive and have higher ID — I take over"',
                logs: [
                    { node: 'n2', msg: '[bully] sending OK to Node 3 — I have higher ID', type: 'info' },
                    { node: 'n3', msg: '[bully] received OK from Node 2 — stepping back', type: '' }
                ],
                nodeUpdates: [{ id: 3, state: 'active', label: 'Stepped back' }],
                arrows: [{ from: 2, to: 3, label: 'OK', color: 'green' }]
            },
            {
                title: 'Node 2 sends ELECTION to Node 1 (no response)',
                detail: 'Node 2 tries higher-ID nodes — Node 1 is DOWN, times out',
                logs: [
                    { node: 'n2', msg: '[bully] sending ELECTION to Node 1 (higher ID check)', type: 'info' },
                    { node: 'n2', msg: '[bully] Node 1 did not respond — timeout 3000ms', type: 'warn' }
                ],
                nodeUpdates: [{ id: 2, state: 'election', label: 'Electing' }],
                arrows: [{ from: 2, to: 1, label: 'ELECTION', color: 'orange' }]
            },
            {
                title: 'Node 2 declares itself Leader',
                detail: 'No higher node responded → Node 2 wins Bully election',
                logs: [
                    { node: 'n2', msg: '[bully] ★ I am the new LEADER — clock=6', type: 'success' },
                ],
                nodeUpdates: [{ id: 2, state: 'leader', label: 'Leader', icon: '👑', clock: 6 }],
                arrows: []
            },
            {
                title: 'Node 2 broadcasts COORDINATOR message',
                detail: 'Informs all alive nodes of the new leader',
                logs: [
                    { node: 'n2', msg: '[bully] broadcasting COORDINATOR(leader=2) to peers', type: 'info' },
                    { node: 'n3', msg: '[bully] received COORDINATOR — Node 2 is new leader', type: '' }
                ],
                nodeUpdates: [{ id: 3, state: 'active', label: 'Follower', clock: 7 }],
                arrows: [{ from: 2, to: 3, label: 'COORDINATOR', color: 'yellow' }]
            },
            {
                title: 'Election complete — system stable',
                detail: 'Node 2 leads. Node 3 follows. Node 1 remains DOWN.',
                logs: [
                    { node: 'sys', msg: '─────────────────────────────────', type: '' },
                    { node: 'sys', msg: 'ELECTION DONE: Leader = Node 2', type: 'success' },
                    { node: 'sys', msg: 'Active nodes: [2, 3]  |  Down: [1]', type: '' }
                ],
                nodeUpdates: [
                    { id: 2, state: 'leader', label: 'Leader', icon: '👑' },
                    { id: 3, state: 'active', label: 'Follower', icon: '🖥️' }
                ],
                arrows: []
            }
        ]
    },

    // ──────────────────────────────────────────────────────────────────
    // 2. NORMAL BOOKING (RA MUTEX)
    // ──────────────────────────────────────────────────────────────────
    booking: {
        meta: {
            icon: '🎫',
            title: 'Normal Seat Booking (Ricart–Agrawala)',
            desc: 'Client requests seat → Node 2 acquires distributed mutex via RA → books → releases'
        },
        init: [
            { id: 1, state: 'active', label: 'Follower', icon: '🖥️', clock: 2 },
            { id: 2, state: 'leader', label: 'Leader', icon: '👑', clock: 4 },
            { id: 3, state: 'active', label: 'Follower', icon: '🖥️', clock: 3 }
        ],
        steps: [{
                title: 'Client sends booking request to Leader (Node 2)',
                detail: 'POST /api/book  {user:"Alice", seat:"A1"}',
                logs: [
                    { node: 'sys', msg: 'HTTP POST /api/book {"user":"Alice","seat":"A1"}', type: 'info' },
                    { node: 'n2', msg: '[booking] received request — seat=A1 user=Alice', type: '' }
                ],
                nodeUpdates: [{ id: 2, state: 'highlight', label: 'Request rcvd' }],
                arrows: []
            },
            {
                title: 'Node 2 increments Lamport clock, sends RA REQUEST',
                detail: 'clock++ → 5  |  REQUEST(seat=A1, ts=5) → Node 1, Node 3',
                logs: [
                    { node: 'n2', msg: '[lamport] clock: 4 → 5', type: '' },
                    { node: 'n2', msg: '[ra] broadcasting REQUEST(ts=5, seat=A1) to all', type: 'info' }
                ],
                nodeUpdates: [{ id: 2, state: 'election', label: 'Requesting', clock: 5 }],
                arrows: [
                    { from: 2, to: 1, label: 'REQUEST(ts=5)', color: 'blue' },
                    { from: 2, to: 3, label: 'REQUEST(ts=5)', color: 'blue' }
                ]
            },
            {
                title: 'Node 1 receives REQUEST, no conflict — replies OK',
                detail: 'Node 1 has no pending request → grants immediately',
                logs: [
                    { node: 'n1', msg: '[ra] received REQUEST(ts=5) from Node 2', type: '' },
                    { node: 'n1', msg: '[ra] no conflict — sending REPLY to Node 2', type: 'info' },
                    { node: 'n1', msg: '[lamport] clock: 2 → max(2,5)+1 = 6', type: '' }
                ],
                nodeUpdates: [{ id: 1, state: 'highlight', label: 'Granting…', clock: 6 }],
                arrows: [{ from: 1, to: 2, label: 'REPLY', color: 'green' }]
            },
            {
                title: 'Node 3 receives REQUEST, no conflict — replies OK',
                detail: 'Node 3 has no pending request → grants immediately',
                logs: [
                    { node: 'n3', msg: '[ra] received REQUEST(ts=5) from Node 2', type: '' },
                    { node: 'n3', msg: '[ra] no conflict — sending REPLY to Node 2', type: 'info' },
                    { node: 'n3', msg: '[lamport] clock: 3 → max(3,5)+1 = 6', type: '' }
                ],
                nodeUpdates: [{ id: 3, state: 'highlight', label: 'Granting…', clock: 6 }],
                arrows: [{ from: 3, to: 2, label: 'REPLY', color: 'green' }]
            },
            {
                title: 'Node 2 received all REPLYs — enters Critical Section',
                detail: 'replies=2/2 → mutex acquired → booking seat A1',
                logs: [
                    { node: 'n2', msg: '[ra] all replies received (2/2) — entering critical section', type: 'success' },
                    { node: 'n2', msg: '[booking] LOCKING seat A1 — writing to state store', type: '' }
                ],
                nodeUpdates: [
                    { id: 2, state: 'cs', label: 'Crit. Section' },
                    { id: 1, state: 'active', label: 'Waiting' },
                    { id: 3, state: 'active', label: 'Waiting' }
                ],
                arrows: []
            },
            {
                title: 'Seat A1 booked successfully',
                detail: 'seat_store["A1"] = {user:"Alice", ts:5, node:2}',
                logs: [
                    { node: 'n2', msg: '[booking] ✓ seat A1 BOOKED for Alice (clock=5)', type: 'success' },
                    { node: 'n2', msg: '[booking] HTTP 200 → {"status":"booked","seat":"A1"}', type: 'success' }
                ],
                nodeUpdates: [{ id: 2, state: 'leader', label: 'Booked ✓', clock: 6 }],
                arrows: []
            },
            {
                title: 'Node 2 sends RELEASE to all nodes',
                detail: 'Exits critical section — notifies peers to unblock',
                logs: [
                    { node: 'n2', msg: '[ra] sending RELEASE — exiting critical section', type: 'info' }
                ],
                nodeUpdates: [{ id: 2, state: 'leader', label: 'Leader' }],
                arrows: [
                    { from: 2, to: 1, label: 'RELEASE', color: 'purple' },
                    { from: 2, to: 3, label: 'RELEASE', color: 'purple' }
                ]
            },
            {
                title: 'Booking flow complete — seat A1 confirmed',
                detail: 'All nodes updated. Mutex released. Dashboard refreshed.',
                logs: [
                    { node: 'n1', msg: '[ra] received RELEASE from Node 2 — mutex free', type: '' },
                    { node: 'n3', msg: '[ra] received RELEASE from Node 2 — mutex free', type: '' },
                    { node: 'sys', msg: '─────────────────────────────────', type: '' },
                    { node: 'sys', msg: 'BOOKING COMPLETE: Alice → seat A1', type: 'success' }
                ],
                nodeUpdates: [
                    { id: 1, state: 'active', label: 'Follower' },
                    { id: 3, state: 'active', label: 'Follower' }
                ],
                arrows: []
            }
        ]
    },

    // ──────────────────────────────────────────────────────────────────
    // 3. CONCURRENT BOOKING CONFLICT
    // ──────────────────────────────────────────────────────────────────
    conflict: {
        meta: {
            icon: '⚡',
            title: 'Concurrent Booking Conflict (RA Priority)',
            desc: 'Node 1 (ts=3) and Node 3 (ts=7) both want seat B2 — Lamport timestamp decides winner'
        },
        init: [
            { id: 1, state: 'active', label: 'Follower', icon: '🖥️', clock: 3 },
            { id: 2, state: 'leader', label: 'Leader', icon: '👑', clock: 6 },
            { id: 3, state: 'active', label: 'Follower', icon: '🖥️', clock: 7 }
        ],
        steps: [{
                title: 'Node 1 & Node 3 simultaneously request seat B2',
                detail: 'Node1: REQUEST(B2,ts=3)  |  Node3: REQUEST(B2,ts=7)',
                logs: [
                    { node: 'n1', msg: '[ra] REQUEST(seat=B2, ts=3) — sending to peers', type: 'info' },
                    { node: 'n3', msg: '[ra] REQUEST(seat=B2, ts=7) — sending to peers', type: 'info' }
                ],
                nodeUpdates: [
                    { id: 1, state: 'election', label: 'Requesting' },
                    { id: 3, state: 'election', label: 'Requesting' }
                ],
                arrows: []
            },
            {
                title: 'Node 1 sends REQUEST to Node 2 & Node 3',
                detail: 'REQUEST(ts=3, seat=B2) → all peers',
                logs: [
                    { node: 'n1', msg: '[ra] → Node 2: REQUEST(B2, ts=3)', type: '' },
                    { node: 'n1', msg: '[ra] → Node 3: REQUEST(B2, ts=3)', type: '' }
                ],
                nodeUpdates: [],
                arrows: [
                    { from: 1, to: 2, label: 'REQUEST(ts=3)', color: 'orange' },
                    { from: 1, to: 3, label: 'REQUEST(ts=3)', color: 'orange' }
                ]
            },
            {
                title: 'Node 3 sends REQUEST to Node 1 & Node 2',
                detail: 'REQUEST(ts=7, seat=B2) → all peers',
                logs: [
                    { node: 'n3', msg: '[ra] → Node 1: REQUEST(B2, ts=7)', type: '' },
                    { node: 'n3', msg: '[ra] → Node 2: REQUEST(B2, ts=7)', type: '' }
                ],
                nodeUpdates: [],
                arrows: [
                    { from: 3, to: 1, label: 'REQUEST(ts=7)', color: 'red' },
                    { from: 3, to: 2, label: 'REQUEST(ts=7)', color: 'red' }
                ]
            },
            {
                title: 'Node 3 receives Node 1\'s request — Node 1 wins (ts=3 < ts=7)',
                detail: 'Timestamps compared: 3 < 7 → Node 1 has priority → Node 3 DEFERS',
                logs: [
                    { node: 'n3', msg: '[ra] conflict on B2: my ts=7, Node 1 ts=3', type: 'warn' },
                    { node: 'n3', msg: '[ra] Node 1 wins (lower timestamp) — deferring my REPLY', type: 'warn' },
                    { node: 'n2', msg: '[ra] Node 1 ts=3 < Node 3 ts=7 — sending REPLY to Node 1', type: '' }
                ],
                nodeUpdates: [{ id: 3, state: 'active', label: 'Deferred' }],
                arrows: [{ from: 2, to: 1, label: 'REPLY', color: 'green' }]
            },
            {
                title: 'Node 1 collects all REPLYs — enters Critical Section',
                detail: 'Replies from Node 2 ✓ + Node 3 already queued ✓ → mutex acquired',
                logs: [
                    { node: 'n1', msg: '[ra] all replies received — entering critical section', type: 'success' },
                    { node: 'n1', msg: '[booking] LOCKING seat B2', type: '' }
                ],
                nodeUpdates: [{ id: 1, state: 'cs', label: 'Crit. Section' }],
                arrows: []
            },
            {
                title: 'Node 1 books seat B2 successfully',
                detail: 'seat_store["B2"] = {user:"Bob", ts:3, node:1}',
                logs: [
                    { node: 'n1', msg: '[booking] ✓ seat B2 BOOKED for Bob (ts=3)', type: 'success' },
                    { node: 'n1', msg: '[ra] sending RELEASE to all nodes', type: 'info' }
                ],
                nodeUpdates: [{ id: 1, state: 'active', label: 'Booked ✓' }],
                arrows: [
                    { from: 1, to: 2, label: 'RELEASE', color: 'purple' },
                    { from: 1, to: 3, label: 'RELEASE', color: 'purple' }
                ]
            },
            {
                title: 'Node 3 now gets its deferred reply — enters CS',
                detail: 'RELEASE received → Node 3 deferred reply now sent → Node 3 enters CS',
                logs: [
                    { node: 'n3', msg: '[ra] RELEASE from Node 1 — sending my deferred REPLY', type: '' },
                    { node: 'n3', msg: '[ra] all replies now received — entering critical section', type: 'success' },
                    { node: 'n3', msg: '[booking] checking seat B2…', type: '' }
                ],
                nodeUpdates: [{ id: 3, state: 'cs', label: 'Crit. Section' }],
                arrows: [{ from: 3, to: 1, label: 'REPLY (deferred)', color: 'green' }]
            },
            {
                title: 'Node 3 finds seat B2 already booked — conflict resolved',
                detail: 'seat_store["B2"].booked = true → returns SEAT_UNAVAILABLE',
                logs: [
                    { node: 'n3', msg: '[booking] seat B2 already booked by Node 1', type: 'error' },
                    { node: 'n3', msg: '[booking] HTTP 409 → {"status":"unavailable","seat":"B2"}', type: 'error' },
                    { node: 'n3', msg: '[ra] sending RELEASE — exiting empty-handed', type: '' }
                ],
                nodeUpdates: [{ id: 3, state: 'active', label: 'Rejected' }],
                arrows: [
                    { from: 3, to: 1, label: 'RELEASE', color: 'purple' },
                    { from: 3, to: 2, label: 'RELEASE', color: 'purple' }
                ]
            },
            {
                title: 'Conflict resolution complete — consistency maintained',
                detail: 'B2 booked once, exactly once. No double-booking. RA mutex guaranteed safety.',
                logs: [
                    { node: 'sys', msg: '─────────────────────────────────', type: '' },
                    { node: 'sys', msg: 'RESULT: B2 → Bob (Node 1, ts=3)', type: 'success' },
                    { node: 'sys', msg: 'CONFLICT RESOLVED — no double-booking', type: 'success' }
                ],
                nodeUpdates: [
                    { id: 1, state: 'active', label: 'Follower' },
                    { id: 3, state: 'active', label: 'Follower' }
                ],
                arrows: []
            }
        ]
    },

    // ──────────────────────────────────────────────────────────────────
    // 4. NODE FAILURE & RECOVERY
    // ──────────────────────────────────────────────────────────────────
    heartbeat: {
        meta: {
            icon: '💔',
            title: 'Node Failure & Heartbeat Recovery',
            desc: 'Node 3 crashes mid-operation → peers detect timeout → Bully election → system recovers'
        },
        init: [
            { id: 1, state: 'active', label: 'Follower', icon: '🖥️', clock: 4 },
            { id: 2, state: 'leader', label: 'Leader', icon: '👑', clock: 8 },
            { id: 3, state: 'active', label: 'Follower', icon: '🖥️', clock: 6 }
        ],
        steps: [{
                title: 'All nodes exchanging heartbeats normally',
                detail: 'interval=2000ms — PING/PONG between all nodes',
                logs: [
                    { node: 'n1', msg: '[heartbeat] PING → Node 2 ✓  PING → Node 3 ✓', type: '' },
                    { node: 'n2', msg: '[heartbeat] PING → Node 1 ✓  PING → Node 3 ✓', type: '' },
                    { node: 'n3', msg: '[heartbeat] PING → Node 1 ✓  PING → Node 2 ✓', type: '' }
                ],
                nodeUpdates: [],
                arrows: [
                    { from: 2, to: 1, label: '♥ PING', color: 'green' },
                    { from: 2, to: 3, label: '♥ PING', color: 'green' }
                ]
            },
            {
                title: 'Node 3 crashes unexpectedly',
                detail: 'SIGKILL received — process terminated — all connections dropped',
                logs: [
                    { node: 'n3', msg: 'FATAL: out of memory — process killed', type: 'error' },
                    { node: 'sys', msg: 'Node 3 is now DOWN (unexpected crash)', type: 'error' }
                ],
                nodeUpdates: [{ id: 3, state: 'down', label: 'CRASHED', icon: '💀' }],
                arrows: []
            },
            {
                title: 'Node 1 & Node 2 detect timeout from Node 3',
                detail: 'No PONG for 5000ms — marking Node 3 as unavailable',
                logs: [
                    { node: 'n1', msg: '[heartbeat] timeout — Node 3 unreachable (5010ms)', type: 'warn' },
                    { node: 'n2', msg: '[heartbeat] timeout — Node 3 unreachable (5018ms)', type: 'warn' }
                ],
                nodeUpdates: [
                    { id: 1, state: 'election', label: 'Detecting…' },
                    { id: 2, state: 'election', label: 'Detecting…' }
                ],
                arrows: []
            },
            {
                title: 'Node 1 starts Bully Election (no response from Node 3)',
                detail: 'Node 1 sends ELECTION to higher-ID nodes (only Node 2)',
                logs: [
                    { node: 'n1', msg: '[bully] starting election — sending ELECTION to Node 2', type: 'info' }
                ],
                nodeUpdates: [],
                arrows: [{ from: 1, to: 2, label: 'ELECTION', color: 'orange' }]
            },
            {
                title: 'Node 2 responds OK — takes over election',
                detail: 'Node 2 has higher ID → responds OK, starts own election to Node 3 (no answer)',
                logs: [
                    { node: 'n2', msg: '[bully] sending OK to Node 1 — I have higher ID', type: 'info' },
                    { node: 'n2', msg: '[bully] sending ELECTION to Node 3 — timeout 3000ms', type: 'warn' },
                    { node: 'n2', msg: '[bully] Node 3 not responding — declaring myself leader', type: 'success' }
                ],
                nodeUpdates: [{ id: 2, state: 'leader', label: 'Leader', icon: '👑' }],
                arrows: [{ from: 2, to: 1, label: 'OK', color: 'green' }]
            },
            {
                title: 'Node 2 broadcasts COORDINATOR message',
                detail: 'COORDINATOR(leader=2) → Node 1   (Node 3 is DOWN, unreachable)',
                logs: [
                    { node: 'n2', msg: '[bully] broadcasting COORDINATOR(leader=2)', type: 'info' },
                    { node: 'n1', msg: '[bully] COORDINATOR received — Node 2 is leader', type: '' }
                ],
                nodeUpdates: [{ id: 1, state: 'active', label: 'Follower', clock: 9 }],
                arrows: [{ from: 2, to: 1, label: 'COORDINATOR', color: 'yellow' }]
            },
            {
                title: 'System continues operating with 2 nodes',
                detail: 'Bookings proceed. Node 3 marked as failed peer in both remaining nodes.',
                logs: [
                    { node: 'n2', msg: '[peer] Node 3 removed from active peer list', type: 'warn' },
                    { node: 'n1', msg: '[peer] Node 3 removed from active peer list', type: 'warn' },
                    { node: 'sys', msg: 'Cluster: 2/3 nodes active — system operational', type: 'success' }
                ],
                nodeUpdates: [
                    { id: 2, state: 'leader', label: 'Leader' },
                    { id: 1, state: 'active', label: 'Follower' }
                ],
                arrows: []
            },
            {
                title: 'Node 3 restarts and rejoins cluster',
                detail: 'On startup, Node 3 connects to peers — immediately follows Node 2',
                logs: [
                    { node: 'n3', msg: '[startup] node restarted — connecting to peers', type: 'info' },
                    { node: 'n3', msg: '[bully] Node 2 is current leader — following', type: '' },
                    { node: 'sys', msg: 'Cluster: 3/3 nodes active — FULLY RECOVERED', type: 'success' }
                ],
                nodeUpdates: [{ id: 3, state: 'active', label: 'Recovered', icon: '🖥️', clock: 10 }],
                arrows: [{ from: 3, to: 2, label: 'HELLO', color: 'blue' }]
            }
        ]
    },

    // ──────────────────────────────────────────────────────────────────
    // 5. LAMPORT CLOCK SYNC
    // ──────────────────────────────────────────────────────────────────
    lamport: {
        meta: {
            icon: '🕐',
            title: 'Lamport Logical Clock Synchronization',
            desc: 'Messages carry timestamps — receivers sync: clock = max(local, received) + 1'
        },
        init: [
            { id: 1, state: 'active', label: 'clock=1', icon: '🖥️', clock: 1 },
            { id: 2, state: 'active', label: 'clock=5', icon: '🖥️', clock: 5 },
            { id: 3, state: 'active', label: 'clock=3', icon: '🖥️', clock: 3 }
        ],
        steps: [{
                title: 'Initial state: each node has its own local clock',
                detail: 'Node1=1  Node2=5  Node3=3  (clocks are not synchronised)',
                logs: [
                    { node: 'n1', msg: '[lamport] initial clock = 1', type: '' },
                    { node: 'n2', msg: '[lamport] initial clock = 5', type: '' },
                    { node: 'n3', msg: '[lamport] initial clock = 3', type: '' }
                ],
                nodeUpdates: [
                    { id: 1, state: 'highlight', label: 'clock=1' },
                    { id: 2, state: 'highlight', label: 'clock=5' },
                    { id: 3, state: 'highlight', label: 'clock=3' }
                ],
                arrows: []
            },
            {
                title: 'Node 2 sends a booking request to Node 1',
                detail: 'Before sending: Node 2 increments clock: 5 → 6',
                logs: [
                    { node: 'n2', msg: '[lamport] sending: clock++ → 6', type: 'info' },
                    { node: 'n2', msg: '[lamport] → Node 1 msg with ts=6', type: '' }
                ],
                nodeUpdates: [{ id: 2, state: 'election', label: 'Sending (6)', clock: 6 }],
                arrows: [{ from: 2, to: 1, label: 'msg(ts=6)', color: 'blue' }]
            },
            {
                title: 'Node 1 receives message — syncs clock',
                detail: 'clock = max(local=1, received=6) + 1 = 7',
                logs: [
                    { node: 'n1', msg: '[lamport] received msg with ts=6', type: '' },
                    { node: 'n1', msg: '[lamport] clock = max(1, 6) + 1 = 7', type: 'success' }
                ],
                nodeUpdates: [{ id: 1, state: 'highlight', label: 'clock=7', clock: 7 }],
                arrows: []
            },
            {
                title: 'Node 1 forwards event to Node 3',
                detail: 'Before sending: clock++ → 8  |  sends to Node 3 with ts=8',
                logs: [
                    { node: 'n1', msg: '[lamport] sending: clock++ → 8', type: 'info' },
                    { node: 'n1', msg: '[lamport] → Node 3 msg with ts=8', type: '' }
                ],
                nodeUpdates: [{ id: 1, state: 'election', label: 'Sending (8)', clock: 8 }],
                arrows: [{ from: 1, to: 3, label: 'msg(ts=8)', color: 'orange' }]
            },
            {
                title: 'Node 3 receives message — syncs clock',
                detail: 'clock = max(local=3, received=8) + 1 = 9',
                logs: [
                    { node: 'n3', msg: '[lamport] received msg with ts=8', type: '' },
                    { node: 'n3', msg: '[lamport] clock = max(3, 8) + 1 = 9', type: 'success' }
                ],
                nodeUpdates: [{ id: 3, state: 'highlight', label: 'clock=9', clock: 9 }],
                arrows: []
            },
            {
                title: 'Node 3 replies back to Node 2',
                detail: 'clock++ → 10  |  REPLY sent to Node 2 with ts=10',
                logs: [
                    { node: 'n3', msg: '[lamport] sending reply: clock++ → 10', type: 'info' },
                    { node: 'n3', msg: '[lamport] → Node 2 REPLY with ts=10', type: '' }
                ],
                nodeUpdates: [{ id: 3, state: 'election', label: 'Sending (10)', clock: 10 }],
                arrows: [{ from: 3, to: 2, label: 'REPLY(ts=10)', color: 'green' }]
            },
            {
                title: 'Node 2 receives REPLY — final clock state',
                detail: 'clock = max(6, 10) + 1 = 11  |  Causal ordering preserved!',
                logs: [
                    { node: 'n2', msg: '[lamport] received REPLY with ts=10', type: '' },
                    { node: 'n2', msg: '[lamport] clock = max(6, 10) + 1 = 11', type: 'success' },
                    { node: 'sys', msg: '─────────────────────────────────', type: '' },
                    { node: 'sys', msg: 'FINAL: N1=8  N2=11  N3=10', type: 'success' },
                    { node: 'sys', msg: 'Causal order maintained: e1 → e2 → e3', type: 'success' }
                ],
                nodeUpdates: [
                    { id: 1, state: 'active', label: 'clock=8' },
                    { id: 2, state: 'active', label: 'clock=11', clock: 11 },
                    { id: 3, state: 'active', label: 'clock=10' }
                ],
                arrows: []
            }
        ]
    }
};

// ═══════════════════════════════════════════════════════════════════════
// STATE
// ═══════════════════════════════════════════════════════════════════════
let currentScenario = 'bully';
let currentStep = -1;
let isPlaying = false;
let playTimer = null;
let activeArrowEls = [];
let startTime = null;

// Node X positions (fraction of container, used for SVG arrows)
const NODE_X_FRACTIONS = { 1: 0.16, 2: 0.50, 3: 0.84 };
const ARROW_BASE_Y = 60; // px from top of diagram — start of lifelines
const ARROW_STEP_H = 48; // px per step

// ═══════════════════════════════════════════════════════════════════════
// INIT
// ═══════════════════════════════════════════════════════════════════════
function loadScenario(id) {
    if (isPlaying) stopPlay();

    // Update sidebar active state
    document.querySelectorAll('.scenario-btn').forEach(b => b.classList.toggle('active', b.dataset.id === id));

    currentScenario = id;
    currentStep = -1;
    const sc = SCENARIOS[id];

    // Update header
    document.getElementById('sc-icon').textContent = sc.meta.icon;
    document.getElementById('sc-title').textContent = sc.meta.title;
    document.getElementById('sc-desc').textContent = sc.meta.desc;

    // Reset nodes
    sc.init.forEach(n => {
        setNodeBox(n.id, n.state, n.label, n.icon || '🖥️', n.clock || 0);
    });

    // Clear arrows
    clearArrows();

    // Render empty step list
    renderStepList(sc.steps);

    // Reset log
    document.getElementById('log-entries').innerHTML = '<div style="color:var(--muted);padding:8px;font-size:0.8rem">Click ▶ Run Demo to start…</div>';

    // Reset progress
    setProgress(0, sc.steps.length);

    // Reset play button
    setPlayBtn(false);

    // Hide done banner
    document.getElementById('done-banner').classList.remove('show');

    // Dot idle
    setLiveDot(false);
}

function init() {
    loadScenario('bully');
}

// ═══════════════════════════════════════════════════════════════════════
// NODE VISUALS
// ═══════════════════════════════════════════════════════════════════════
const STATE_CLASSES = ['state-leader', 'state-active', 'state-down', 'state-election', 'state-cs', 'state-highlight'];

function setNodeBox(id, state, label, icon, clock) {
    const box = document.getElementById(`nbox-${id}`);
    const badge = document.getElementById(`nbadge-${id}`);
    const clk = document.getElementById(`nclock-${id}`);
    const life = document.getElementById(`nlife-${id}`);

    STATE_CLASSES.forEach(c => box.classList.remove(c));
    box.classList.add(`state-${state}`);

    // icon
    box.querySelector('.node-icon').textContent = icon || '🖥️';
    // label inside box
    box.querySelector('.node-label').textContent = label;

    // badge
    badge.textContent = label;
    const BADGE_MAP = { leader: 'badge-leader', active: 'badge-active', down: 'badge-down', election: 'badge-election', cs: 'badge-cs', highlight: 'badge-active' };
    ['badge-leader', 'badge-active', 'badge-down', 'badge-election', 'badge-cs'].forEach(c => badge.classList.remove(c));
    if (BADGE_MAP[state]) badge.classList.add(BADGE_MAP[state]);

    // clock
    if (clock !== undefined) {
        clk.textContent = `clock: ${clock}`;
        clk.classList.add('changed');
        setTimeout(() => clk.classList.remove('changed'), 800);
    }

    // lifeline
    if (life) life.classList.toggle('down', state === 'down');
}

// ═══════════════════════════════════════════════════════════════════════
// STEPS LIST
// ═══════════════════════════════════════════════════════════════════════
function renderStepList(steps) {
    const el = document.getElementById('steps-list');
    el.innerHTML = steps.map((s, i) => `
    <div class="step-item" id="step-item-${i}">
      <div class="step-num">${i + 1}</div>
      <div class="step-body">
        <div class="step-title">${s.title}</div>
        <div class="step-detail">${s.detail}</div>
      </div>
    </div>
  `).join('');
}

function highlightStep(index, steps) {
    steps.forEach((_, i) => {
        const el = document.getElementById(`step-item-${i}`);
        if (!el) return;
        el.classList.remove('active', 'done');
        if (i < index) el.classList.add('done');
        if (i === index) {
            el.classList.add('active');
            el.scrollIntoView({ behavior: 'smooth', block: 'nearest' });
        }
    });
}

// ═══════════════════════════════════════════════════════════════════════
// ARROWS (SVG)
// ═══════════════════════════════════════════════════════════════════════
function clearArrows() {
    const svg = document.getElementById('msg-svg');
    svg.innerHTML = '';
    activeArrowEls = [];
}

function drawArrow(fromNode, toNode, label, color, stepIndex) {
    const diag = document.getElementById('nodes-diagram');
    const w = diag.offsetWidth;
    const x1 = w * NODE_X_FRACTIONS[fromNode];
    const x2 = w * NODE_X_FRACTIONS[toNode];
    const y = ARROW_BASE_Y + stepIndex * ARROW_STEP_H;

    const svg = document.getElementById('msg-svg');
    const colorVal = {
        green: '#3fb950',
        blue: '#58a6ff',
        orange: '#ffa657',
        red: '#f85149',
        purple: '#a371f7',
        yellow: '#e3b341'
    }[color] || '#58a6ff';

    // Determine arrow direction
    const dir = x2 > x1 ? 1 : -1;
    const arrowLen = Math.abs(x2 - x1) - 12;
    const startX = x1 + dir * 6;
    const endX = x2 - dir * 6;

    // Line
    const line = document.createElementNS('http://www.w3.org/2000/svg', 'line');
    line.setAttribute('x1', startX);
    line.setAttribute('y1', y);
    line.setAttribute('x2', endX);
    line.setAttribute('y2', y);
    line.setAttribute('stroke', colorVal);
    line.setAttribute('stroke-width', '2');
    line.setAttribute('stroke-dasharray', '6 3');
    line.setAttribute('opacity', '0');
    line.style.transition = 'opacity 0.3s';

    // Arrowhead
    const ah = document.createElementNS('http://www.w3.org/2000/svg', 'polygon');
    const tipX = endX;
    const tipY = y;
    const pts = dir === 1 ?
        `${tipX},${tipY} ${tipX-8},${tipY-4} ${tipX-8},${tipY+4}` :
        `${tipX},${tipY} ${tipX+8},${tipY-4} ${tipX+8},${tipY+4}`;
    ah.setAttribute('points', pts);
    ah.setAttribute('fill', colorVal);
    ah.setAttribute('opacity', '0');
    ah.style.transition = 'opacity 0.3s';

    // Label
    const txt = document.createElementNS('http://www.w3.org/2000/svg', 'text');
    txt.setAttribute('x', (startX + endX) / 2);
    txt.setAttribute('y', y - 6);
    txt.setAttribute('fill', colorVal);
    txt.setAttribute('font-size', '10');
    txt.setAttribute('text-anchor', 'middle');
    txt.setAttribute('font-family', 'monospace');
    txt.setAttribute('opacity', '0');
    txt.style.transition = 'opacity 0.3s 0.2s';
    txt.textContent = label;

    // Background pill for label
    const bgRect = document.createElementNS('http://www.w3.org/2000/svg', 'rect');
    const tlen = label.length * 6;
    bgRect.setAttribute('x', (startX + endX) / 2 - tlen / 2 - 3);
    bgRect.setAttribute('y', y - 18);
    bgRect.setAttribute('width', tlen + 6);
    bgRect.setAttribute('height', 14);
    bgRect.setAttribute('rx', '3');
    bgRect.setAttribute('fill', '#0d1117');
    bgRect.setAttribute('opacity', '0');
    bgRect.style.transition = 'opacity 0.3s 0.15s';

    svg.appendChild(bgRect);
    svg.appendChild(line);
    svg.appendChild(ah);
    svg.appendChild(txt);

    // Animate in
    requestAnimationFrame(() => {
        requestAnimationFrame(() => {
            [line, ah, txt, bgRect].forEach(el => el.setAttribute('opacity', el === bgRect ? '0.9' : '1'));
        });
    });

    activeArrowEls.push(line, ah, txt, bgRect);
}

// ═══════════════════════════════════════════════════════════════════════
// LOG
// ═══════════════════════════════════════════════════════════════════════
function addLogs(logs) {
    const container = document.getElementById('log-entries');
    const now = new Date();
    const ts = `${String(now.getHours()).padStart(2,'0')}:${String(now.getMinutes()).padStart(2,'0')}:${String(now.getSeconds()).padStart(2,'0')}`;

    logs.forEach((log, idx) => {
        const entry = document.createElement('div');
        entry.className = 'log-entry';
        entry.innerHTML = `
      <span class="log-time">${ts}</span>
      <span class="log-node ${log.node}">[${log.node.toUpperCase()}]</span>
      <span class="log-msg ${log.type || ''}">${log.msg}</span>
    `;
        container.appendChild(entry);

        setTimeout(() => {
            entry.classList.add('show');
            container.scrollTop = container.scrollHeight;
        }, idx * 80);
    });
}

// ═══════════════════════════════════════════════════════════════════════
// PROGRESS
// ═══════════════════════════════════════════════════════════════════════
function setProgress(step, total) {
    const pct = total > 0 ? Math.round((step / total) * 100) : 0;
    document.getElementById('progress-fill').style.width = `${pct}%`;
    document.getElementById('progress-label').textContent = `${step} / ${total} steps`;
}

function setPlayBtn(playing) {
    const btn = document.getElementById('play-btn');
    btn.textContent = playing ? '⏸ Pause' : '▶ Run Demo';
    btn.style.background = playing ? 'var(--yellow)' : 'var(--green)';
}

function setLiveDot(active) {
    document.getElementById('live-dot').classList.toggle('active', active);
    document.getElementById('live-label').textContent = active ? 'LIVE' : 'IDLE';
}

// ═══════════════════════════════════════════════════════════════════════
// PLAYBACK ENGINE
// ═══════════════════════════════════════════════════════════════════════
function playScenario() {
    if (isPlaying) {
        stopPlay();
        return;
    }

    const sc = SCENARIOS[currentScenario];
    if (currentStep >= sc.steps.length - 1) {
        // restart from beginning
        loadScenario(currentScenario);
    }

    isPlaying = true;
    setPlayBtn(true);
    setLiveDot(true);
    document.getElementById('done-banner').classList.remove('show');

    advanceStep();
}

function advanceStep() {
    const sc = SCENARIOS[currentScenario];
    currentStep++;

    if (currentStep >= sc.steps.length) {
        // Done
        stopPlay();
        document.getElementById('done-banner').classList.add('show');
        document.getElementById('done-banner').textContent = `✅ Scenario Complete: "${sc.meta.title}"`;
        return;
    }

    const step = sc.steps[currentStep];

    // Highlight step
    highlightStep(currentStep, sc.steps);

    // Node updates
    if (step.nodeUpdates && step.nodeUpdates.length > 0) {
        // fetch current node info for any unspecified fields
        step.nodeUpdates.forEach(nu => {
            const box = document.getElementById(`nbox-${nu.id}`);
            const currentIcon = box.querySelector('.node-icon').textContent;
            setNodeBox(nu.id, nu.state, nu.label, nu.icon || currentIcon, nu.clock);
        });
    }

    // Arrows
    if (step.arrows && step.arrows.length > 0) {
        step.arrows.forEach((a, ai) => {
            setTimeout(() => drawArrow(a.from, a.to, a.label, a.color, currentStep), ai * 200);
        });
    }

    // Logs
    if (step.logs && step.logs.length > 0) {
        addLogs(step.logs);
    }

    // Progress
    setProgress(currentStep + 1, sc.steps.length);

    // Schedule next
    const delay = parseInt(document.getElementById('speed-select').value, 10) || 2000;
    playTimer = setTimeout(advanceStep, delay);
}

function stopPlay() {
    isPlaying = false;
    if (playTimer) { clearTimeout(playTimer);
        playTimer = null; }
    setPlayBtn(false);
    setLiveDot(false);
}

function resetScenario() {
    stopPlay();
    loadScenario(currentScenario);
}

// ═══════════════════════════════════════════════════════════════════════
// KEYBOARD SHORTCUT
// ═══════════════════════════════════════════════════════════════════════
document.addEventListener('keydown', e => {
    if (e.code === 'Space' && e.target === document.body) {
        e.preventDefault();
        playScenario();
    }
    if (e.code === 'KeyR' && e.target === document.body) {
        resetScenario();
    }
    const keys = ['Digit1', 'Digit2', 'Digit3', 'Digit4', 'Digit5'];
    const ids = ['bully', 'booking', 'conflict', 'heartbeat', 'lamport'];
    const ki = keys.indexOf(e.code);
    if (ki !== -1) loadScenario(ids[ki]);
});

// ═══════════════════════════════════════════════════════════════════════
// BOOT
// ═══════════════════════════════════════════════════════════════════════
window.addEventListener('load', init);