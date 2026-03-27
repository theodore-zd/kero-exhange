function getAccessToken() {
    return localStorage.getItem('accessToken');
}

function setAccessToken(token) {
    localStorage.setItem('accessToken', token);
}

function clearAccessToken() {
    localStorage.removeItem('accessToken');
}

function getWalletUUID() {
    return localStorage.getItem('walletUUID');
}

function setWalletUUID(uuid) {
    localStorage.setItem('walletUUID', uuid);
}

function clearWalletUUID() {
    localStorage.removeItem('walletUUID');
}

function signOut() {
    clearAccessToken();
    clearWalletUUID();
    window.location.href = '/signin';
}

function authenticatedFetch(url, options = {}) {
    const token = getAccessToken();
    if (!token) {
        signOut();
        return Promise.reject(new Error('No access token'));
    }
    
    const headers = {
        ...options.headers,
        'Authorization': `Bearer ${token}`
    };
    
    return fetch(url, { ...options, headers })
        .then(response => {
            if (response.status === 401) {
                signOut();
                return Promise.reject(new Error('Unauthorized'));
            }
            return response;
        });
}
