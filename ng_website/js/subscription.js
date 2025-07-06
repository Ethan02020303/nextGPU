let currentOrderId = null;
let pollingInterval = null;
const TOTAL_TIMEOUT = 300;

document.addEventListener('DOMContentLoaded', async function() {

    // 显示用户信息
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
            document.querySelectorAll('.faq-answer').forEach(ans => {
                ans.style.display = 'none';
            });
            document.querySelectorAll('.faq-question i').forEach(icon => {
                icon.className = 'fas fa-chevron-down';
            });
            if (!isOpen) {
                answer.style.display = 'block';
                question.querySelector('i').className = 'fas fa-chevron-up';
            }
        });
    });


    // 获取按钮和模态框元素
    const primaryBtn = document.getElementById('primaryPlanBtn');
    const intermediateBtn = document.getElementById('intermediatePlanBtn');
    const paymentModal = document.getElementById('paymentModal');
    const closeModal = document.getElementById('closeModal');
    const tabButtons = document.querySelectorAll('.tab-button');
    const paymentApp = document.getElementById('paymentApp');
    const planName = document.getElementById('planName');
    const paymentAmount = document.getElementById('paymentAmount');
    const paymentTimer = document.getElementById('paymentTimer');
    const paymentContainer = document.getElementById('paymentContainer');
    const qrImage = document.getElementById('qrImage');
    const qrLoader = document.getElementById('qrLoader');
    const subscriptionPlans = {
        primary: {
            name: "初级会员",
            price: 29.00
        },
        intermediate: {
            name: "中级会员",
            price: 99.00
        }
    };
    let currentPlan = null;
    let currentPlanId = null;
    let countdown = null;
    let paymentMethod = 'wechat';
    
    // 打开模态框函数
    function openPaymentModal(planId) {
        currentPlanId = planId;
        currentPlan = subscriptionPlans[planId];
        planName.textContent = currentPlan.name;
        paymentAmount.textContent = `¥${currentPlan.price.toFixed(2)}`;
        paymentModal.style.display = 'flex';
        document.querySelector('.tab-button[data-tab="wechat"]').classList.add('active');
        document.querySelector('.tab-button[data-tab="alipay"]').classList.remove('active');
        paymentApp.textContent = '微信';
        paymentContainer.style.display = 'block';
        startPaymentTimer(TOTAL_TIMEOUT); 
        getPaymentQRCode(planId, paymentMethod);
    }
    
    // 启动支付倒计时
    function startPaymentTimer(seconds) {
        let timeLeft = seconds;
        if (countdown) {
            clearInterval(countdown);
        }
        updateTimerDisplay(timeLeft);
        countdown = setInterval(() => {
            timeLeft--;
            updateTimerDisplay(timeLeft);
            
            if (timeLeft <= 0) {
                clearInterval(countdown);
                showExpiredMessage();
            }
        }, 1000);
    }

    // 更新倒计时显示
    function updateTimerDisplay(seconds) {
        const minutes = Math.floor(seconds / 60);
        const remainingSeconds = seconds % 60;
        paymentTimer.textContent = `${minutes.toString().padStart(2, '0')}:${remainingSeconds.toString().padStart(2, '0')}`;
        if (seconds <= 30) {
            paymentTimer.style.color = '#e74c3c';
        } else {
            paymentTimer.style.color = '#495057';
        }
    }
    
    // 显示支付过期消息
    function showExpiredMessage() {
        const expiredMessage = document.createElement('div');
        expiredMessage.className = 'expired-message';
        expiredMessage.textContent = '支付已过期，请重新操作';
        paymentContainer.innerHTML = '';
        paymentContainer.appendChild(expiredMessage);
    }
    
    // 模拟支付成功
    function simulatePaymentSuccess() {
        clearInterval(countdown);
        paymentContainer.style.display = 'none';
        setTimeout(() => {
            paymentModal.style.display = 'none';
            alert('订阅成功！您的账户已升级为' + currentPlan.name);
        }, 3000);
    }
    
    // 获取支付二维码
    let existingQRCode = null; // 用于跟踪现有的二维码实例
    async function getPaymentQRCode(planId, method) {
        const qrcodeDiv = document.getElementById('qrcode');
        const qrLoader = document.getElementById('qrLoader');
        if (existingQRCode) {
            existingQRCode.clear(); 
            qrcodeDiv.innerHTML = ''; 
            existingQRCode = null;
        }
        qrLoader.style.display = 'flex';

        const qrData = await sendSubscription(userName, planId);
        if(qrData.codeId === 200) {
            existingQRCode = new QRCode(qrcodeDiv, {
                text: qrData.qrcode,
                width: 200,
                height: 200,
                colorDark: "#000000",
                colorLight: "#ffffff",
                correctLevel: QRCode.CorrectLevel.H
            });
            currentOrderId = qrData.orderID;
            startPaymentStatusPolling();
            qrLoader.style.display = 'none';
        }else{
            throw new Error(qrData.msg);
        }
    }

    // 新增停止轮询函数
    function stopPolling() {
        clearInterval(pollingInterval);
        pollingInterval = null;
        pollingAttempts = 0;
    }

    // 新增错误提示函数
    function showError(message) {
        const paymentModal = document.getElementById('paymentModal');
        if (paymentModal) {
            paymentModal.style.display = 'none';
            alert(message);
        }
    }

    // 修改支付成功处理
    function handlePaymentSuccess() {
        alert('订阅成功！您的账户已升级为' + currentPlan.name);
        window.location.reload(); 
    }

    // 轮询订单状态
    function startPaymentStatusPolling() {
        if (pollingInterval) clearInterval(pollingInterval);
        startTime = Date.now();


        pollingInterval = setInterval(async () => {

            // 检查是否超时
            if (Date.now() - startTime > TOTAL_TIMEOUT * 1000) {
                showError('支付超时，请重新操作');
                stopPolling();
                return;
            }

            try {
                const data = await sendCheckSubscription(currentOrderId);
                if(data.codeId === 200) {
                    switch(data.status) {
                        case 'SUCCESS':
                            clearInterval(pollingInterval);
                            handlePaymentSuccess();

                            break;
                        case 'FAILED':
                            showError('支付失败，请重新尝试');
                            stopPolling();
                            break;
                        case 'CLOSED':
                        case 'REVOKED':
                            showError('支付已过期');
                            stopPolling();
                            break;
                        default:
                            break;
                    }
                }else{
                    showError('支付状态查询失败');
                    stopPolling();
                }
            } catch (error) {
                console.error('轮询异常:', error);
                showError('网络异常，请稍后重试');
                stopPolling();
            }
        }, 3000); 
    }

    // 切换支付方式
    function switchPaymentMethod(method) {
        paymentMethod = method;
        paymentApp.textContent = method === 'wechat' ? '微信' : '支付宝';
        if (currentPlanId) {
            getPaymentQRCode(currentPlanId, method);
        }
    }
    
    // 事件监听
    primaryBtn.addEventListener('click', () => openPaymentModal('primary'));
    intermediateBtn.addEventListener('click', () => openPaymentModal('intermediate'));
    document.addEventListener('keydown', (event) => {
        if (event.key === 's' && paymentModal.style.display === 'flex') {
            simulatePaymentSuccess();
        }
    });
    
    closeModal.addEventListener('click', () => {
        paymentModal.style.display = 'none';
        clearInterval(countdown);
        if (pollingInterval) {
            clearInterval(pollingInterval);
            pollingInterval = null;
        }
        // document.querySelector('.tab-content').classList.remove('active');
    });
    
    // 关闭模态框（点击外部区域）
    window.addEventListener('click', (event) => {
        if (event.target === paymentModal) {
            paymentModal.style.display = 'none';
            clearInterval(countdown);
            if (pollingInterval) {
                clearInterval(pollingInterval);
                pollingInterval = null;
            }
            // document.querySelector('.tab-content').classList.remove('active');
        }
    });
    
    // 切换支付方式标签
    tabButtons.forEach(button => {
        button.addEventListener('click', () => {
            const tab = button.dataset.tab;
            tabButtons.forEach(btn => btn.classList.remove('active'));
            button.classList.add('active');
            switchPaymentMethod(tab);
        });
    });
    
    // 模拟支付成功按钮（仅用于演示）
    document.addEventListener('keydown', (event) => {
        if (event.key === 's' && paymentModal.style.display === 'flex') {
            simulatePaymentSuccess();
        }
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

