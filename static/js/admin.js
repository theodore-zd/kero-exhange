function getAdminToken() { return localStorage.getItem('adminToken'); }
function setAdminToken(token) { localStorage.setItem('adminToken', token); }

function adminSignOut() {
    console.info('[admin] signing out, clearing token');
    localStorage.removeItem('adminToken');
    window.location.href = '/admin/login';
}

async function adminFetch(url, opts) {
    opts = opts || {};
    var token = getAdminToken();
    if (!token) {
        console.warn('[admin] no auth token found, redirecting to login');
        adminSignOut();
        return null;
    }
    var headers = Object.assign({ 'Authorization': 'Bearer ' + token }, opts.headers || {});
    if (opts.body && typeof opts.body === 'object') {
        headers['Content-Type'] = 'application/json';
        opts.body = JSON.stringify(opts.body);
    }
    var method = (opts.method || 'GET').toUpperCase();
    console.info('[admin] ' + method + ' ' + url);
    try {
        var resp = await fetch(url, Object.assign({}, opts, { headers: headers }));
        if (resp.status === 401) {
            console.warn('[admin] 401 unauthorized from ' + url + ', signing out');
            adminSignOut();
            return null;
        }
        if (!resp.ok) {
            var body;
            try { body = await resp.clone().json(); } catch (_) { body = null; }
            console.error('[admin] ' + method + ' ' + url + ' failed',
                'status=' + resp.status,
                'body=', body);
        }
        return resp;
    } catch (err) {
        console.error('[admin] ' + method + ' ' + url + ' network error:', err.message || err);
        throw err;
    }
}

/**
 * Show an error message in the page error div with console detail.
 * @param {string} containerId - ID of the error div.
 * @param {string} userMessage - Message shown in the UI.
 * @param {*} [detail] - Extra detail logged to console (error object, response body, etc).
 */
function showError(containerId, userMessage, detail) {
    var el = document.getElementById(containerId);
    if (el) {
        el.textContent = userMessage;
        el.style.display = '';
    }
    if (detail) {
        console.error('[admin] ' + userMessage, detail);
    } else {
        console.error('[admin] ' + userMessage);
    }
}

/**
 * Parse and unwrap a successful adminFetch response.
 * Logs and returns null if the response is missing or not ok.
 * @param {Response|null} resp - The fetch response.
 * @param {string} label - Label for log messages (e.g. "wallets").
 * @returns {Promise<object|null>} The parsed .data from the response envelope, or null.
 */
async function parseResponse(resp, label) {
    if (!resp) {
        console.warn('[admin] ' + label + ': no response (auth redirect?)');
        return null;
    }
    if (!resp.ok) {
        var errBody;
        try { errBody = await resp.json(); } catch (_) { errBody = null; }
        var msg = (errBody && errBody.error && errBody.error.message) || 'request failed (status ' + resp.status + ')';
        console.error('[admin] ' + label + ': ' + msg, errBody);
        return { _error: msg };
    }
    var json;
    try {
        json = await resp.json();
    } catch (err) {
        console.error('[admin] ' + label + ': failed to parse JSON response:', err.message);
        return { _error: 'invalid response from server' };
    }
    if (!json.data) {
        console.warn('[admin] ' + label + ': response has no .data field', json);
    }
    console.info('[admin] ' + label + ': ok');
    return json.data || {};
}
