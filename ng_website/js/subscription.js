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
    // 为导航菜单添加活动状态切换
    const navItems = document.querySelectorAll('.nav-item');
    navItems.forEach(item => {
        item.addEventListener('click', function() {
            navItems.forEach(i => i.classList.remove('active'));
            this.classList.add('active');
        });
    });
    
    // 为操作按钮添加悬停效果
    const actionButtons = document.querySelectorAll('.action-btn');
    actionButtons.forEach(btn => {
        btn.addEventListener('mouseenter', function() {
            this.style.boxShadow = '0 10px 25px rgba(123, 104, 238, 0.25)';
        });
        
        btn.addEventListener('mouseleave', function() {
            this.style.boxShadow = '0 8px 20px rgba(123, 104, 238, 0.15)';
        });
    });
    
    // 为订阅卡片添加简单动画
    const planCards = document.querySelectorAll('.plan-card');
    planCards.forEach(card => {
        card.addEventListener('mouseenter', function() {
            if(!this.classList.contains('pro')) {
                this.style.transform = 'translateY(-5px)';
                this.style.boxShadow = '0 12px 30px rgba(123, 104, 238, 0.2)';
            }
        });
        
        card.addEventListener('mouseleave', function() {
            if(!this.classList.contains('pro')) {
                this.style.transform = 'none';
                this.style.boxShadow = '0 8px 20px rgba(123, 104, 238, 0.15)';
            }
        });
    });

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
            price: 39.00
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
        paymentMethod = 'wechat';
        paymentContainer.style.display = 'block';
        // startPaymentTimer(TOTAL_TIMEOUT); 
        switchPaymentMethod(paymentMethod);
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
    async function getWxPaymentQRCode(planId, method) {
        const qrcodeDiv = document.getElementById('qrcode');
        const qrLoader = document.getElementById('qrLoader');
        if (existingQRCode) {
            existingQRCode.clear(); 
            qrcodeDiv.innerHTML = ''; 
            existingQRCode = null;
        }
        qrLoader.style.display = 'flex';
        const qrData = await sendWxSubscription(userName, planId);
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
            startPaymentStatusPolling(1);
            qrLoader.style.display = 'none';
        }else{
            throw new Error(qrData.msg);
        }
    }
	async function getAliPaymentQRCode(planId, method) {
        const wechatContainer = document.getElementById('wechatQrContainer');
        const alipayContainer = document.getElementById('alipayIframeContainer');
        const alipayIframe = document.getElementById('alipayIframe');
        const alipayLoader = document.getElementById('alipayLoader');
        const qrLoader = document.getElementById('qrLoader');
        
        try {
            // 清除微信二维码
            if (existingQRCode) {
                existingQRCode.clear();
                document.getElementById('qrcode').innerHTML = '';
                existingQRCode = null;
            }
            
            // 切换显示
            wechatContainer.style.display = 'none';
            alipayContainer.style.display = 'block';
            alipayLoader.style.display = 'flex';
            qrLoader.style.display = 'none';
            
            // 获取支付宝支付URL
            const qrData = await sendAliSubscription(userName, planId);
            
            if(qrData.codeId === 200) {
                // 设置iframe的src为支付宝返回的支付页面URL
                alipayIframe.src = qrData.qrcode;
                currentOrderId = qrData.orderID;
                
                // 监听iframe加载完成
                alipayIframe.onload = function() {
                    alipayLoader.style.display = 'none';
                    // 确保iframe居中
                    alipayIframe.style.display = 'block';
                    alipayIframe.style.margin = '0 auto';
                    startPaymentStatusPolling(2);
                };
                
                // 监听iframe加载错误
                alipayIframe.onerror = function() {
                    alipayLoader.style.display = 'none';
                    showError('支付宝支付页面加载失败，请重试');
                    wechatContainer.style.display = 'block';
                };
            } else {
                throw new Error(qrData.msg || '支付宝支付创建失败');
            }
        } catch (error) {
            alipayLoader.style.display = 'none';
            wechatContainer.style.display = 'block';
            showError(error.message);
            throw error;
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
    function startPaymentStatusPolling(mode) {
        if (pollingInterval) clearInterval(pollingInterval);
        stopPolling();
        startTime = Date.now();
        pollingInterval = setInterval(async () => {
            if (Date.now() - startTime > TOTAL_TIMEOUT * 1000) {
                showError('支付超时，请重新操作');
                stopPolling();
                return;
            }
            try {
                if(mode === 1) {    //微信
                    const data = await sendWxCheckSubscription(currentOrderId);
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
                }else if (mode === 2) { //支付宝
                    const data = await sendAliCheckSubscription(currentOrderId);
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
        const wechatContainer = document.getElementById('wechatQrContainer');
        const alipayContainer = document.getElementById('alipayIframeContainer');
        paymentApp.textContent = method === 'wechat' ? '微信' : '支付宝';

		if (paymentApp.textContent === '微信') {
            wechatContainer.style.display = 'block';
            alipayContainer.style.display = 'none';
            startPaymentTimer(TOTAL_TIMEOUT); 
            if (currentPlanId) {
                getWxPaymentQRCode(currentPlanId, method);
            }
		}else if (paymentApp.textContent === '支付宝') {
			wechatContainer.style.display = 'none';
            alipayContainer.style.display = 'block';
            startPaymentTimer(TOTAL_TIMEOUT); 
            if (currentPlanId) {
                getAliPaymentQRCode(currentPlanId, method);
            }
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
        stopPolling();
        if (pollingInterval) {
            clearInterval(pollingInterval);
            pollingInterval = null;
        }
    });
    
    // 关闭模态框（点击外部区域）
    window.addEventListener('click', (event) => {
        if (event.target === paymentModal) {
            paymentModal.style.display = 'none';
            stopPolling();
            clearInterval(countdown);
            if (pollingInterval) {
                clearInterval(pollingInterval);
                pollingInterval = null;
            }
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

