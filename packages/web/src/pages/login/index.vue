<template>
  <section
    class="w-screen h-screen flex *:m-auto bg-linear-to-t from-[#BFA4A0] to-[#7784AC] "
  >
    <section class="w-full max-w-sm flex flex-col gap-10 ">
      <section>
        <img
          src="../../../public/logo.png"
          width="100"
          alt="logo.png"
          class="m-auto"
        >
        <h3 class="scroll-m-20 text-2xl font-semibold tracking-tight text-white text-center">
          Memoh
        </h3>
      </section>

      <Card class="py-14">
        <CardContent>
          <form>
            <div class="grid w-full items-center gap-4 [&_input]:py-5">
              <div class="flex flex-col space-y-1.5 gap-2">
                <Label
                  for="email"
                  class=""
                >Email</Label>
                <Input
                  id="email"
                  type="email"                
                  placeholder="m@example.com"
                />
              </div>
              <div class="flex flex-col space-y-1.5 gap-2">
                <div class="flex items-center ">
                  <Label for="password">Password</Label>
                </div>
                <Input
                  id="password"
                  type="password"
                />
              </div>
            </div>
            <div class="flex">
              <a
                href="#"
                class="ml-auto inline-block text-sm underline mt-2"
              >
                Forgot your password?
              </a>
            </div>
          </form>
        </CardContent>
        <CardFooter class="flex flex-col gap-4">
          <Button
            class="w-full"
            @click="login"
          >
            登录
          </Button>
          <Button
            variant="outline"
            class="w-full"
          >
            注册
          </Button>
        </CardFooter>
      </Card>
    </section>
  </section>
</template>

<script setup lang="ts">
import {
  Card,
  CardContent,
  CardFooter,
  Input,
  Label,
  Button
} from '@memoh/ui'
import { useRouter } from 'vue-router'


const router = useRouter()

const login = async () => {
  // 先模拟一下数据
  localStorage.setItem('token','afewfewf')
  await router.push('/main')
  console.log('登录')
}





</script>