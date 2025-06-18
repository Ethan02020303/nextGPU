
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

