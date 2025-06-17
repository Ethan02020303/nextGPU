
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

// 添加自动执行功能
document.addEventListener('DOMContentLoaded', function() {
    // 检查是否存在用户信息区域
    if (document.getElementById('user-avatar') && document.getElementById('user-name')) {
        const currentUser = localStorage.getItem('currentUser');
        sendUserInfo(currentUser);
        setInterval(() => {
            fetchUserInfo(currentUser);
        }, 60 * 1000);
    }
});