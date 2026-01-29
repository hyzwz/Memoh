import { defineStore } from 'pinia'
import { reactive } from 'vue'


type user={
  'id': string,
  'username': string,
  'role': string,
  'displayName': string
}
export const useUserStore = defineStore('user', () => {
  const userInfo = reactive<user>({
    'id': '',
    'username': '',
    'role': '',
    'displayName': ''
  })


  const login = (userData: user,token:string) => {
    localStorage.setItem('token',token)
    for (const key of Object.keys(userData) as (keyof user)[]) {
      userInfo[key] = userData[key]
    }
  }

  const exitLogin = () => {
    localStorage.removeItem('token')
    for (const key of Object.keys(userInfo) as (keyof user)[]) {
      userInfo[key]=''
    }
  }
  return {
    userInfo,
    login,
    exitLogin
  }
}, {
  persist:true
})
