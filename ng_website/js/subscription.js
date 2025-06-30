document.addEventListener('DOMContentLoaded', function() {

    const {logined, userName} = checkLoginStatus();
    if(logined){
        sendUserInfo(userName)
    }else{
        sendUserInfo()
    }

    // 常见问题折叠功能
    const faqQuestions = document.querySelectorAll('.faq-question');
    faqQuestions.forEach(question => {
        question.addEventListener('click', () => {
            const answer = question.nextElementSibling;
            const isOpen = answer.style.display === 'block';
            
            // 关闭所有答案
            document.querySelectorAll('.faq-answer').forEach(ans => {
                ans.style.display = 'none';
            });
            
            // 重置所有箭头
            document.querySelectorAll('.faq-question i').forEach(icon => {
                icon.className = 'fas fa-chevron-down';
            });
            
            // 切换当前答案
            if (!isOpen) {
                answer.style.display = 'block';
                question.querySelector('i').className = 'fas fa-chevron-up';
            }
        });
    });
});

// 交互效果：卡片悬停动画
document.querySelectorAll('.plan-card').forEach(card => {
    card.addEventListener('mouseenter', function() {
        this.style.transform = 'translateY(-10px)';
        this.style.boxShadow = '0 15px 40px rgba(0, 0, 0, 0.15)';
    });
    
    card.addEventListener('mouseleave', function() {
        this.style.transform = 'translateY(0)';
        this.style.boxShadow = '0 10px 30px rgba(0, 0, 0, 0.1)';
    });
});

// 禁用按钮状态处理
document.querySelector('.btn-disabled').addEventListener('click', function(e) {
    e.preventDefault();
    alert('此会员级别为免费体验版，无需购买');
});

// 为其他按钮添加购买提示
document.querySelectorAll('.btn:not(.btn-disabled)').forEach(btn => {
    btn.addEventListener('click', function() {
        const planCard = this.closest('.plan-card');
        const planId = planCard.dataset.planId;
        sendSubscription(planId);
    });
});