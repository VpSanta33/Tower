import request from './request'

// 获取 JSFinder 全局配置
export function getJSFinderConfig() {
  return request.post('/jsfinder/config/get')
}

// 保存 JSFinder 全局配置
export function saveJSFinderConfig(data) {
  return request.post('/jsfinder/config/save', data)
}

// 重置为内置默认值
export function resetJSFinderConfig() {
  return request.post('/jsfinder/config/reset')
}
