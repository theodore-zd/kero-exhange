function requireAuth() {
    if (!localStorage.getItem('accessToken')) {
        window.location.href = '/signin';
        return false;
    }
    return true;
}

function getAccessToken() { return localStorage.getItem('accessToken'); }
function setAccessToken(token) { localStorage.setItem('accessToken', token); }
function getWalletUUID() { return localStorage.getItem('walletUUID'); }
function setWalletUUID(uuid) { localStorage.setItem('walletUUID', uuid); }

function signOut() {
    localStorage.removeItem('accessToken');
    localStorage.removeItem('walletUUID');
    window.location.href = '/signin';
}

function showError(containerId, message) {
    var el = document.getElementById(containerId);
    if (el) {
        el.textContent = message;
        el.style.display = '';
    }
}

async function apiFetch(url, opts) {
    opts = opts || {};
    var token = getAccessToken();
    if (!token) { signOut(); return null; }
    var headers = Object.assign({ 'Authorization': 'Bearer ' + token }, opts.headers || {});
    if (opts.body && typeof opts.body === 'object') {
        headers['Content-Type'] = 'application/json';
        opts.body = JSON.stringify(opts.body);
    }
    var resp = await fetch(url, Object.assign({}, opts, { headers: headers }));
    if (resp.status === 401) { signOut(); return null; }
    return resp;
}

async function parseResponse(resp, label) {
    if (!resp) return null;
    if (!resp.ok) {
        var errBody;
        try { errBody = await resp.json(); } catch (_) { errBody = null; }
        var msg = (errBody && errBody.error && errBody.error.message) || 'request failed';
        return { _error: msg };
    }
    var json;
    try {
        json = await resp.json();
    } catch (err) {
        return { _error: 'invalid response from server' };
    }
    return json.data || {};
}
