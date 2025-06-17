// 全局变量
let socket = null;
const maxReconnectAttempts = 5; // 最大重连次数
let reconnectAttempts = 0;
let reconnectTimer = null;

// 接收消息时派发自定义事件
function handleOnReadMessage(data) {
  // 创建自定义事件
  const messageEvent = new CustomEvent('onReadMessage', {
    detail: { message: data }
  });
  
  // 派发事件到全局window对象
  window.dispatchEvent(messageEvent);
}

function handleOnDisconnectMessage(data) {
  // 创建自定义事件
  const messageEvent = new CustomEvent('onDisconnectMessage', {
    detail: { message: data }
  });
  
  // 派发事件到全局window对象
  window.dispatchEvent(messageEvent);
}

function handleOnErrorMessage(data) {
  // 创建自定义事件
  const messageEvent = new CustomEvent('onErrorMessage', {
    detail: { message: data }
  });
  
  // 派发事件到全局window对象
  window.dispatchEvent(messageEvent);
}

// 初始化WebSocket连接
function connect(wsUrl) {

    if(socket !== null){
      return
    }

    socket = new WebSocket(wsUrl);

    //链接服务端
    socket.onopen = () => {
        reconnectAttempts = 0; // 重置重连计数器
        console.log('WebSocket连接已成功建立');
    };
    
    //接收到数据
    socket.onmessage = (event) => {
        console.log(`收到消息: ${event.data}`)
        try {
            const data = JSON.parse(event.data);
            // 调用消息处理函数
            handleOnReadMessage(data);
            } catch (error) {
            console.error('消息解析错误:', error);
        }
    };

    // 连接关闭时
    socket.onclose = (event) => {
        const message = event.wasClean 
            ? `连接已正常关闭 (code=${event.code}, reason=${event.reason})`
            : `连接意外断开`;
        // handleOnDisconnectMessage(data)
        // 尝试自动重连 (仅在非正常关闭且未达到重连上限时)
        if (!event.wasClean && reconnectAttempts < maxReconnectAttempts) {
            const delay = Math.pow(2, reconnectAttempts) * 1000; // 指数退避策略            
            reconnectTimer = setTimeout(() => {
                reconnectAttempts++;
            }, delay);
        }
    };

    // 错误处理
    socket.onerror = (error) => {
        console.error('WebSocket错误:', error);
        socket.close();
        socket = null;
        handleOnErrorMessage();
    };
}