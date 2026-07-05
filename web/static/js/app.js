document.addEventListener('DOMContentLoaded', function() {
    console.log('抢课系统已加载');
    initApp();
});

function initApp() {
    checkLoginStatus();
    setupEventListeners();
}

function checkLoginStatus() {
    fetch('/api/session')
        .then(response => response.json())
        .then(data => {
            if (data.logged_in) {
                updateUIForLoggedInUser();
            }
        })
        .catch(error => {
            console.error('检查登录状态失败:', error);
        });
}

function updateUIForLoggedInUser() {
    const loginBtn = document.querySelector('a[href="/login"]');
    if (loginBtn) {
        loginBtn.textContent = '进入系统';
        loginBtn.href = '/courses';
    }
}

function setupEventListeners() {
    const loginForm = document.getElementById('loginForm');
    if (loginForm) {
        loginForm.addEventListener('submit', handleLogin);
    }
    
    const taskForm = document.getElementById('taskForm');
    if (taskForm) {
        taskForm.addEventListener('submit', handleCreateTask);
    }
}

function handleLogin(event) {
    event.preventDefault();
    
    const username = document.getElementById('username').value;
    const password = document.getElementById('password').value;
    const loginURL = document.getElementById('loginURL').value;
    const captchaId = document.getElementById('captchaId');
    const captchaAnswer = document.getElementById('captchaAnswer');
    
    const loginData = {
        username: username,
        password: password,
        login_url: loginURL
    };
    
    // 如果有验证码字段，添加到请求中
    if (captchaId && captchaAnswer && captchaId.value) {
        loginData.captcha_id = captchaId.value;
        loginData.captcha_answer = captchaAnswer.value;
    }
    
    fetch('/api/login', {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json',
        },
        body: JSON.stringify(loginData)
    })
    .then(response => response.json())
    .then(data => {
        if (data.error) {
            showError(data.error);
            // 如果登录失败，刷新验证码
            if (typeof loadCaptcha === 'function') {
                loadCaptcha();
            }
        } else {
            showSuccess('登录成功');
            setTimeout(() => {
                window.location.href = '/courses';
            }, 1000);
        }
    })
    .catch(error => {
        showError('登录失败: ' + error.message);
    });
}

function handleCreateTask(event) {
    event.preventDefault();
    
    const strategy = document.getElementById('strategy').value;
    const courseCheckboxes = document.querySelectorAll('input[name="courses"]:checked');
    const courseIds = Array.from(courseCheckboxes).map(cb => cb.value);
    
    if (courseIds.length === 0) {
        showError('请至少选择一门课程');
        return;
    }
    
    fetch('/api/tasks', {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json',
        },
        body: JSON.stringify({
            strategy: strategy,
            course_ids: courseIds
        })
    })
    .then(response => response.json())
    .then(data => {
        if (data.error) {
            showError(data.error);
        } else {
            showSuccess('任务创建成功');
            setTimeout(() => {
                window.location.href = '/tasks';
            }, 1000);
        }
    })
    .catch(error => {
        showError('创建任务失败: ' + error.message);
    });
}

function showError(message) {
    const errorDiv = document.createElement('div');
    errorDiv.className = 'error-message';
    errorDiv.textContent = message;
    document.body.appendChild(errorDiv);
    setTimeout(() => {
        errorDiv.remove();
    }, 3000);
}

function showSuccess(message) {
    const successDiv = document.createElement('div');
    successDiv.className = 'success-message';
    successDiv.textContent = message;
    document.body.appendChild(successDiv);
    setTimeout(() => {
        successDiv.remove();
    }, 3000);
}
