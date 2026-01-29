<template>
  <div
    ref="displayContainer"
    class="flex flex-col gap-4"
  >
    <template
      v-for="chatItem in chatList"
      :key="chatItem.id"
    >
      <UserChat
        v-if="chatItem.action === 'user'"
        :user-say="chatItem"
      />
      <RobotChat
        v-if="chatItem.action === 'robot'"
        :robot-say="chatItem"
      />
    </template>
  </div>
</template>

<script setup lang="ts">
import UserChat from './UserChat/index.vue'
import RobotChat from './RobotChat/index.vue'
import { inject, ref, watch } from 'vue'
import { useElementBounding } from '@vueuse/core'
import { useChatList } from '@/store/ChatList'
import { onBeforeRouteLeave } from 'vue-router'
// 模拟一下数据
const {chatList,add} = useChatList()

const chatSay = inject('chatSay', ref(''))
// 模拟一下对话
watch(chatSay, () => {
  if (chatSay.value) {
    add({
      description: chatSay.value,
      time: new Date(),
      action: 'user',
      id: 1
    })
   
    add({
      description: '',
      time: new Date(),
      action: 'robot',
      id: 2,
      type: 'Openai Gpt5',
      state:'thinking'
    })   
    chatSay.value=''
  }
}, {
  immediate: true
})

const displayContainer = ref()
const { height,top } = useElementBounding(displayContainer)

let prevScroll = 0, curScroll = 0, autoScroll = true,cacheScroll=0

watch(top, () => {
  const container = displayContainer.value?.parentElement?.parentElement
  if (height.value === 0) {
    autoScroll = false
    prevScroll = curScroll=0
  }
  if ((container?.scrollHeight - container.clientHeight - container.scrollTop) < 1) {
    autoScroll = true
    prevScroll=curScroll=container.scrollTop
  }  
})

watch(height, (newVal,oldVal) => {
  const container = displayContainer.value?.parentElement?.parentElement
  if (container) {
    curScroll = container.scrollTop
    if (curScroll < prevScroll) {
      autoScroll = false
    }
    prevScroll = curScroll
  }
  if (oldVal === 0 && newVal > container.clientHeight) {   
    container.scrollTo({
      top: cacheScroll,
    }) 
    return
  }  
  if (!(container && (container?.scrollHeight - container.clientHeight - container.scrollTop) < 1)&&autoScroll) {
    container.scrollTo({
      top: container?.scrollHeight - container.clientHeight,
      behavior: 'smooth',
    })
  } 
})

onBeforeRouteLeave(() => {
  const container = displayContainer.value?.parentElement?.parentElement
  if (container) {
    cacheScroll = container.scrollTop  
  }
  
})

</script>