document.addEventListener('DOMContentLoaded', function() {

    //获取登录信息
    const userName = localStorage.getItem('userName');
    if(userName !== "") {
        sendUserInfo(userName)
    }else{
        sendUserInfo()
    }
});