const KeyCloak_URL = 'https://keycloak.local.moojnn.com';
let token;

/**
 * 显示或隐藏用户信息加载状态
 * @param {boolean} show - 是否显示加载状态
 */
function showUserLoading(show) {
    const elements = document.querySelectorAll('.stat-value');
    
    if (show) {
        elements.forEach(el => {
            if (!el.querySelector('.spinner')) {
                const spinner = document.createElement('div');
                spinner.className = 'spinner';
                el.innerHTML = '';
                el.appendChild(spinner);
            }
        });
    } else {
        // 移除加载状态
        elements.forEach(el => {
            const spinner = el.querySelector('.spinner');
            if (spinner) {
                el.removeChild(spinner);
            }
        });
    }
}

// 检查登录状态
function checkLoginStatus() {
    const isLoggedIn = localStorage.getItem('isLoggedIn');
    if (isLoggedIn === 'true') {
        const username = localStorage.getItem('userName');
        return {logined: true, userName: username}
    }
    return false, ""
}

function setLoginStatus(logined, userName) {
    if(logined){
        localStorage.setItem('isLoggedIn', 'true');
        localStorage.setItem('userName', userName);
    }else{
        localStorage.setItem('isLoggedIn', 'false');
        localStorage.setItem('userName', '');
    }
}

/* SSO统一登录 */
// 检查登录状态
function checkLoginStatus() {
    const token = getCookie('kc-access');
    if (!token) {
        window.location.href = 'https://aip.local.moojnn.com/trade/#/login';
        return;
    }
    fetchUserInfo(token)
    .then(userInfo => {
        sendLogin(userInfo.preferred_username, userInfo.preferred_username, token)
    })
    .catch(error => {
        console.error('Failed to get user info:', error);
        window.location.href = 'https://aip.local.moojnn.com/trade/#/login';
    });
}

// 获取cookie
function getCookie(name) {
    const value = `; ${document.cookie}`;
    const parts = value.split(`; ${name}=`);
    if (parts.length === 2) return parts.pop().split(';').shift();
}

// 调用KeyCloak获取用户信息
async function fetchUserInfo(token) {
    const response = await fetch(
        'https://keycloak.local.moojnn.com/realms/aip/protocol/openid-connect/userinfo',
        {
            headers: {
                'Authorization': `Bearer ${token}`
            }
        }
    );
    if (!response.ok) {
        throw new Error('Failed to fetch user info');
    }
    return await response.json();
}