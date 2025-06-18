document.addEventListener('DOMContentLoaded', function() {

    const {logined, userName} = checkLoginStatus();
    if(logined){
        sendUserInfo(userName)
    }else{
        sendUserInfo()
    }
});