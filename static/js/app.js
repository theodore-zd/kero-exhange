function getAccessToken() { return localStorage.getItem('accessToken'); }
function setAccessToken(token) { localStorage.setItem('accessToken', token); }
function getWalletUUID() { return localStorage.getItem('walletUUID'); }
function setWalletUUID(uuid) { localStorage.setItem('walletUUID', uuid); }

function signOut() {
    localStorage.removeItem('accessToken');
    localStorage.removeItem('walletUUID');
    window.location.href = '/signin';
}

async function apiFetch(url, opts = {}) {
    const token = getAccessToken();
    if (!token) { signOut(); return; }
    const headers = { 'Authorization': 'Bearer ' + token, ...opts.headers };
    if (opts.body && typeof opts.body === 'object') {
        headers['Content-Type'] = 'application/json';
        opts.body = JSON.stringify(opts.body);
    }
    const resp = await fetch(url, { ...opts, headers });
    if (resp.status === 401) { signOut(); return; }
    return resp;
}
