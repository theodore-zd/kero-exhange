function getAdminToken() { return localStorage.getItem('adminToken'); }
function setAdminToken(token) { localStorage.setItem('adminToken', token); }

function adminSignOut() {
    localStorage.removeItem('adminToken');
    window.location.href = '/admin/login';
}

async function adminFetch(url, opts = {}) {
    const token = getAdminToken();
    if (!token) { adminSignOut(); return; }
    const headers = { 'Authorization': 'Bearer ' + token, ...opts.headers };
    if (opts.body && typeof opts.body === 'object') {
        headers['Content-Type'] = 'application/json';
        opts.body = JSON.stringify(opts.body);
    }
    const resp = await fetch(url, { ...opts, headers });
    if (resp.status === 401) { adminSignOut(); return; }
    return resp;
}
