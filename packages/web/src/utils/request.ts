import axios, { type AxiosRequestConfig } from 'axios'
import { useRouter } from 'vue-router'

const router=useRouter()
export default (function () {
  const axiosInstance = axios.create({
    baseURL:'http://localhost:7002/'
  })
 
  axiosInstance.interceptors.request.use((config) => {
    return config
  }, (error) => Promise.reject(error))

  axiosInstance.interceptors.response.use((response) => {
 
    return response
  }, (error) => { 
    if (error?.status === 401) {
      router.replace({
        name:'Login'
      })
    }
    return Promise.reject(error)
  })
  return (params: AxiosRequestConfig,isToken=true) => {
  
    return axiosInstance(params)
  }
}())