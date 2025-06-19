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