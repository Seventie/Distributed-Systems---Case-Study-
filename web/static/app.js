// ═══════════════════════════════════════════════════════════
// Distributed Ticket Booking System — Dashboard Frontend
// Vanilla JS — no frameworks, minimal and educational
// ═══════════════════════════════════════════════════════════

// API base URL (same origin)
const API = '';

// ═══ STATE ═══
let currentFilter = 'all';
let eventLog = [];
let selectedSeat = null;
let activeRaStates = {};
const MAX_LOG_ENTRIES = 200;
const MAX_FLOW_ENTRIES = 50;

// ═══ DOM REFERENCES ═══
const dom = {
    nodeId: document.getElementById('node-id'),
    nodeState: document.getElementById('node-state'),
    leaderId: document.getElementById('leader-id'),
    lamportTime: document.getElementById('lamport-time'),
    electionStatus: document.getElementById('election-status'),
    peersList: document.getElementById('peers-list'),
    seatGrid: document.getElementById('seat-grid'),
    availCount: document.getElementById('avail-count'),
    bookedCount: document.getElementById('booked-count'),
    raStates: document.getElementById('ra-states'),
    algorithmFlow: document.getElementById('algorithm-flow'),
    eventLogEl: document.getElementById('event-log'),
    bookingForm: document.getElementById('booking-form'),
    bookingResult: document.getElementById('booking-result'),
    sseStatus: document.getElementById('sse-status'),
    btnElection: document.getElementById('btn-trigger-election'),
    // Analytics
    statRequests: document.getElementById('stat-requests'),
    statSuccess: document.getElementById('stat-success'),
    statFailed: document.getElementById('stat-failed'),
    statElections: document.getElementById('stat-elections'),
    statRA: document.getElementById('stat-ra'),
    statEvents: document.getElementById('stat-events'),
    bookingsPerSeat: document.getElementById('bookings-per-seat'),
};

// ═══ INITIALIZATION ═══
document.addEventListener('DOMContentLoaded', () => {
    // Initial data fetch
    fetchStatus();
    fetchSeats();
    fetchEvents();
    fetchAnalytics();

    // Set up polling (every 2 seconds)
    setInterval(fetchStatus, 2000);
    setInterval(fetchSeats, 3000);
    setInterval(fetchAnalytics, 5000);

    // Set up SSE for real-time events
    setupSSE();

    // Set up event handlers
    dom.bookingForm.addEventListener('submit', handleBooking);
    dom.btnElection.addEventListener('click', handleTriggerElection);

    // Filter buttons
    document.querySelectorAll('.filter-btn').forEach(btn => {
        btn.addEventListener('click', () => {
            document.querySelectorAll('.filter-btn').forEach(b => b.classList.remove('active'));
            btn.classList.add('active');
            currentFilter = btn.dataset.filter;
            renderEventLog();
        });
    });
});

// ═══ API CALLS ═══

async function fetchJSON(url) {
    try {
        const resp = await fetch(API + url);
        if (!resp.ok) throw new Error(`HTTP ${resp.status}`);
        return await resp.json();
    } catch (err) {
        console.error(`API error (${url}):`, err);
        return null;
    }
}

async function fetchStatus() {
    const data = await fetchJSON('/api/status');
    if (!data) return;

    dom.nodeId.textContent = `Node ${data.node_id}`;
    dom.nodeState.textContent = data.state || 'running';
    dom.lamportTime.textContent = data.lamport_time || 0;

    // Leader display
    if (data.leader_id === -1 || data.leader_id === undefined) {
        dom.leaderId.textContent = 'None';
        dom.leaderId.className = 'value badge';
    } else {
        dom.leaderId.textContent = `Node ${data.leader_id}`;
        if (data.is_leader) {
            dom.leaderId.textContent += ' (me)';
        }
        dom.leaderId.className = 'value badge badge-leader';
    }

    // Election status
    dom.electionStatus.textContent = data.election_running ? '⚡ Running' : 'Idle';
    dom.electionStatus.style.color = data.election_running ? '#fbbf24' : '';

    // Peers
    renderPeers(data.peers || []);

    // RA states
    activeRaStates = data.ra_states || {};
    renderRAStates(activeRaStates);
}

async function fetchSeats() {
    const data = await fetchJSON('/api/seats');
    if (!data) return;

    renderSeatGrid(data.seats || []);
    dom.availCount.textContent = data.available_count || 0;
    dom.bookedCount.textContent = data.booked_count || 0;
}

async function fetchEvents() {
    const data = await fetchJSON('/api/events?limit=100');
    if (!data) return;

    eventLog = data.events || [];
    renderEventLog();
}

async function fetchAnalytics() {
    const data = await fetchJSON('/api/analytics');
    if (!data) return;

    dom.statRequests.textContent = data.total_requests || 0;
    dom.statSuccess.textContent = data.successful || 0;
    dom.statFailed.textContent = data.failed || 0;
    dom.statElections.textContent = data.elections || 0;
    dom.statRA.textContent = data.ra_requests || 0;
    dom.statEvents.textContent = data.total_events || 0;

    renderBookingsPerSeat(data.bookings_per_seat || {});
}

// ═══ BOOKING HANDLER ═══

async function handleBooking(e) {
    if (e) e.preventDefault();

    const userName = document.getElementById('user-name').value.trim();
    const seatId = document.getElementById('seat-id').value.trim().toUpperCase();

    if (!userName || !seatId) return;

    dom.bookingResult.classList.remove('hidden', 'success', 'failure');
    dom.bookingResult.textContent = '🔒 Acquiring Distributed Lock...';
    dom.bookingResult.classList.add('success'); 
    dom.bookingResult.classList.remove('hidden');

    // Force prompt update of seat grid to show "waiting"
    fetchSeats();

    try {
        const resp = await fetch(API + '/api/book', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ user_name: userName, seat_id: seatId })
        });
        const data = await resp.json();

        dom.bookingResult.classList.remove('success', 'failure');
        if (data.success) {
            dom.bookingResult.classList.add('success');
            dom.bookingResult.innerHTML = `<span style="font-size:1.2rem">✅</span> ${data.message}`;
            selectedSeat = null; // Clear selection on success
            document.getElementById('seat-id').value = '';
        } else {
            dom.bookingResult.classList.add('failure');
            dom.bookingResult.innerHTML = `<span style="font-size:1.2rem">❌</span> ${data.message}`;
        }
        
        // Refresh seats and analytics
        fetchSeats();
        setTimeout(fetchAnalytics, 500);
    } catch (err) {
        dom.bookingResult.classList.remove('success');
        dom.bookingResult.classList.add('failure');
        dom.bookingResult.textContent = `Error: ${err.message}`;
    }
}

// ═══ ELECTION TRIGGER ═══

async function handleTriggerElection() {
    try {
        await fetch(API + '/api/election', { method: 'POST' });
        dom.btnElection.textContent = '⚡ Election Triggered!';
        setTimeout(() => { dom.btnElection.textContent = 'Trigger Election'; }, 2000);
    } catch (err) {
        console.error('Failed to trigger election:', err);
    }
}

// ═══ SSE (Server-Sent Events) ═══

function setupSSE() {
    const evtSource = new EventSource(API + '/api/stream');

    evtSource.onopen = () => {
        dom.sseStatus.textContent = '● LIVE';
        dom.sseStatus.classList.remove('disconnected');
    };

    evtSource.onmessage = (e) => {
        try {
            const evt = JSON.parse(e.data);
            addEvent(evt);
            addFlowItem(evt);

            // Auto-refresh on important events
            if (evt.event_type.startsWith('BOOKING_') || evt.event_type === 'SEAT_SYNCED' || evt.event_type.startsWith('RA_')) {
                fetchSeats();
                fetchStatus();
            }
            if (evt.event_type.startsWith('ELECTION_') || evt.event_type === 'LEADER_FAILURE') {
                setTimeout(fetchStatus, 300);
            }
        } catch (err) {
            console.error('SSE parse error:', err);
        }
    };

    evtSource.onerror = () => {
        dom.sseStatus.textContent = '● DISCONNECTED';
        dom.sseStatus.classList.add('disconnected');
    };
}

// ═══ RENDER FUNCTIONS ═══

function renderPeers(peers) {
    if (peers.length === 0) {
        dom.peersList.innerHTML = '<p class="muted">No peers connected</p>';
        return;
    }

    dom.peersList.innerHTML = peers.map(p => `
        <span class="peer-badge">
            <span class="dot ${p.alive ? 'alive' : 'dead'}"></span>
            Node ${p.node_id}
            <small style="margin-left:4px; opacity:0.6">${p.alive ? 'online' : 'offline'}</small>
        </span>
    `).join('');
}

function renderSeatGrid(seats) {
    if (seats.length === 0) {
        dom.seatGrid.innerHTML = '<p class="muted">No seats available</p>';
        return;
    }

    dom.seatGrid.innerHTML = seats.map(s => {
        let classes = `seat ${s.status}`;
        if (s.id === selectedSeat) classes += ' selected';
        
        // Check if this node is currently wanting or holding this seat
        const ra = activeRaStates[s.id];
        const raState = ra ? ra.state : null;
        const deferredCount = (ra && ra.deferred_nodes) ? ra.deferred_nodes.length : 0;

        if (raState === 'WANTING' || raState === 'HELD') {
            classes += ' waiting';
            if (raState === 'HELD') classes += ' in-cs';
        }

        const title = s.status === 'booked' ? 
            `Booked by ${s.booked_by}` : 
            (raState ? `Mutex: ${raState}${deferredCount > 0 ? ` (${deferredCount} nodes waiting)` : ''}` : 'Click to select');
        
        return `
            <div class="${classes}" 
                 title="${title}"
                 onclick="selectSeat('${s.id}', '${s.status}')">
                <span>${s.id}</span>
                ${s.status === 'booked' ? '<small class="seat-label">BOOKED</small>' : ''}
                ${deferredCount > 0 ? `<div class="queue-badge" title="${deferredCount} others waiting">${deferredCount}</div>` : ''}
                ${raState === 'WANTING' ? '<div class="ra-indicator wanting"></div>' : ''}
                ${raState === 'HELD' ? '<div class="ra-indicator held"></div>' : ''}
            </div>
        `;
    }).join('');
}

function selectSeat(seatId, status) {
    if (status === 'booked') return;
    
    if (selectedSeat === seatId) {
        selectedSeat = null;
        document.getElementById('seat-id').value = '';
    } else {
        selectedSeat = seatId;
        document.getElementById('seat-id').value = seatId;
        document.getElementById('user-name').focus();
    }
    renderSeatGrid([]); // This is a bit of a hack to force a rerender, better to pass existing data
    fetchSeats(); // Proper refresh
}

function renderRAStates(states) {
    const entries = Object.entries(states);
    if (entries.length === 0) {
        dom.raStates.innerHTML = '<p class="muted">No active mutex requests</p>';
        return;
    }

    dom.raStates.innerHTML = entries.map(([resource, data]) => {
        const state = data.state;
        const deferred = data.deferred_nodes || [];
        const badgeClass = state === 'WANTING' ? 'ra' : 'booking';
        
        return `
            <div class="ra-state-item">
                <div style="display:flex; justify-content:space-between; align-items:center;">
                    <span style="font-weight:700">Seat ${resource}</span>
                    <span class="flow-badge ${badgeClass}">${state}</span>
                </div>
                ${deferred.length > 0 ? `
                    <div class="deferred-list">
                        <span class="muted" style="font-size:0.7rem">Deferred Replies:</span>
                        <div style="display:flex; gap:4px; margin-top:4px">
                            ${deferred.map(id => `<span class="mini-badge">Node ${id}</span>`).join('')}
                        </div>
                    </div>
                ` : ''}
            </div>
        `;
    }).join('');
}

function renderBookingsPerSeat(data) {
    const entries = Object.entries(data);
    if (entries.length === 0) {
        dom.bookingsPerSeat.innerHTML = '<p class="muted">No data yet</p>';
        return;
    }

    const maxCount = Math.max(...entries.map(([_, c]) => c), 1);
    const maxBarHeight = 80;

    dom.bookingsPerSeat.innerHTML = entries
        .sort((a, b) => a[0].localeCompare(b[0]))
        .map(([seat, count]) => `
            <div class="bar-item">
                <span class="bar-value">${count}</span>
                <div class="bar" style="height: ${Math.max(4, (count / maxCount) * maxBarHeight)}px; background: var(--primary-gradient)"></div>
                <span class="bar-label">${seat}</span>
            </div>
        `).join('');
}

// ═══ EVENT LOG ═══

function addEvent(evt) {
    eventLog.push(evt);
    if (eventLog.length > MAX_LOG_ENTRIES) {
        eventLog = eventLog.slice(-MAX_LOG_ENTRIES);
    }
    renderEventLog();
}

function renderEventLog() {
    const filtered = currentFilter === 'all' ?
        eventLog :
        eventLog.filter(e => e.event_type.includes(currentFilter));

    // Show most recent first
    const recent = filtered.slice(-100).reverse();

    if (recent.length === 0) {
        dom.eventLogEl.innerHTML = '<p class="muted">No events matching filter</p>';
        return;
    }

    dom.eventLogEl.innerHTML = recent.map(e => {
        const date = new Date(e.timestamp);
        const time = date.toLocaleTimeString([], { hour12: false, hour: '2-digit', minute: '2-digit', second: '2-digit' });
        return `
            <div class="log-entry ${e.event_type}" style="padding: 10px 16px; margin-bottom: 4px; border-radius: 8px">
                <div style="display:flex; justify-content:space-between; align-items:center;">
                    <span class="log-time" style="font-weight:700; color:var(--info)">${time}</span>
                    <span class="log-lamport" style="font-size:0.75rem">L${e.lamport_time}</span>
                </div>
                <div style="margin-top:4px; display:flex; gap:10px; align-items:flex-start">
                    <span class="badge" style="background:rgba(255,255,255,0.1); font-size:0.65rem; white-space:nowrap">${e.event_type}</span>
                    <span class="log-detail" style="font-size:0.85rem">${escapeHtml(e.details)}</span>
                </div>
            </div>
        `;
    }).join('');
}

// ═══ ALGORITHM FLOW ═══

function addFlowItem(evt) {
    const flowEl = dom.algorithmFlow;

    // Remove "waiting" placeholder
    const placeholder = flowEl.querySelector('.muted');
    if (placeholder) placeholder.remove();

    // Determine badge category
    let category = 'booking';
    if (evt.event_type.startsWith('ELECTION') || evt.event_type === 'LEADER_FAILURE') category = 'election';
    if (evt.event_type.startsWith('RA_')) category = 'ra';
    if (evt.event_type.startsWith('HEARTBEAT')) category = 'heartbeat';
    if (evt.event_type === 'SEAT_SYNCED') category = 'seat';

    // Friendly label
    const labels = {
        'NODE_STARTED': '🚀 Node Online',
        'BOOKING_REQUEST': '📝 Requesting Seat',
        'BOOKING_SUCCESS': '🎉 Booking Confirmed',
        'BOOKING_FAILED': '⚠️ Booking Refused',
        'ELECTION_START': '⚡ Election Start',
        'ELECTION_OK': '👍 Vote: OK',
        'ELECTION_COORDINATOR': '👑 New Leader Active',
        'LEADER_FAILURE': '💀 Leader Offline',
        'RA_REQUEST_SENT': '📤 Mutex Request Sent',
        'RA_REQUEST_RECEIVED': '📥 Mutex Request Received',
        'RA_REPLY_SENT': '↩️ Mutex Permission Given',
        'RA_REPLY_RECEIVED': '✉️ Mutex Permission Received',
        'RA_ENTER_CS': '🔒 Critical Section: LOCKED',
        'RA_EXIT_CS': '🔓 Critical Section: RELEASED',
        'HEARTBEAT_SENT': '💓 Heartbeat Sent',
        'HEARTBEAT_RECEIVED': '💓 Heartbeat Received',
        'SEAT_SYNCED': '🔄 Cluster Sync: Done',
    };

    const label = labels[evt.event_type] || evt.event_type;
    const time = new Date(evt.timestamp).toLocaleTimeString();

    const html = `
        <div class="flow-item">
            <span class="flow-badge ${category}">${category}</span>
            <span class="flow-text" style="font-weight:600">${label}</span>
            <span class="flow-time" style="opacity:0.6">@ L${evt.lamport_time}</span>
        </div>
    `;

    flowEl.insertAdjacentHTML('afterbegin', html);

    // Trim old entries
    while (flowEl.children.length > MAX_FLOW_ENTRIES) {
        flowEl.removeChild(flowEl.lastChild);
    }
}

// ═══ UTILS ═══

function escapeHtml(text) {
    if (!text) return '';
    const div = document.createElement('div');
    div.textContent = text;
    return div.innerHTML;
}