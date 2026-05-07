import request from './request'

export function login(data) {
  return request.post('/login', data)
}

export function getUserList(data) {
  return request.post('/user/list', data)
}

export function createUser(data) {
  return request.post('/user/create', data)
}

export function updateUser(data) {
  return request.post('/user/update', data)
}

export function deleteUser(data) {
  return request.post('/user/delete', data)
}

export function resetUserPassword(data) {
  return request.post('/user/resetPassword', data)
}

// 首次登录密码重置（不需要原密码验证）
export function firstLoginResetPassword(data) {
  return request.post('/user/firstLoginResetPassword', data)
}
