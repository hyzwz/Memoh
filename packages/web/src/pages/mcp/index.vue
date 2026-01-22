<template>
  <section class="[&_td:last-child]:w-40">
    <CreateMCP />
    <DataTable
      :columns="columns"
      :data="mcpFormatData"
    />
  </section>
</template>

<script setup lang="ts">
import { useQuery,useMutation,useQueryCache } from '@pinia/colada'
import request from '@/utils/request'
import { watch, h, provide,ref, computed,reactive } from 'vue'
import DataTable from '@/components/DataTable/index.vue'
import CreateMCP from '@/components/CreateMCP/index.vue'
import { type ColumnDef } from '@tanstack/vue-table'
import {
  Badge,
  Button
} from '@memoh/ui'
import { type MCPListItem as  MCPType } from '@memoh/shared'


const open = ref(false)
const editMCPData = ref<{
  name: string,
  config: MCPType['config'],
  active: boolean
  id:string
}|null>(null)
provide('open', open)
provide('mcpEditData',editMCPData)

const queryCache=useQueryCache()
const { mutate:DeleteMCP}= useMutation({
  mutation: (id:string) => request({
    url: `/mcp/${id}`,
    method:'DELETE'
  }),
  onSettled() {
    queryCache.invalidateQueries({
      key:['mcp']
    })
  }
})
const columns:ColumnDef<MCPType>[] = [
  {
    accessorKey: 'name',
    header: () => h('div', { class: 'text-left py-4' }, 'Name'),
   
  },
  {
    accessorKey: 'type',
    header: () => h('div', { class: 'text-left' }, 'Type'),
  },
  {
    accessorKey: 'config.command',
    header: () => h('div', { class: 'text-left' }, 'Command'),
  },
  {
    accessorKey: 'config.cwd',
    header: () => h('div', { class: 'text-left' }, 'Cwd'),
  },
  {
    accessorKey: 'config.args',
    header: () => h('div', { class: 'text-left' }, 'Arguments'),
    cell: ({ row }) => h('div', {class:'flex gap-4'}, row.original.config.args.map((argTxt) => {
      return h(Badge, {
        variant:'default'
      },()=>argTxt)
    }))
  },
  {
    accessorKey: 'config.env',
    header: () => h('div', { class: 'text-left' }, 'Env'),
    cell: ({ row }) => h('div', { class: 'flex gap-4' }, Object.entries(row.original.config.env).map(([key,value]) => {
      return h(Badge, {
        variant: 'outline'
      }, ()=>`${key}:${value}`)
    }))
  },
  {
    accessorKey: 'control',
    header: () => h('div', { class: 'text-center' }, '操作'),
    cell: ({ row }) => h('div', {class:'flex gap-2'}, [
      h(Button, {
        onClick() {
          editMCPData.value = {
            name: row.original.name,
            config: {...row.original.config},
            active: row.original.active,
            id:row.original.id
          }       
          open.value=true
        }
      }, ()=>'编辑'),
      h(Button, {
        variant: 'destructive',
        async onClick() {        
          try {
            await DeleteMCP(row.original.id)
          } catch {
            return
          }
        }
      },()=>'删除')
    ])
  }
]

const { data: mcpData } = useQuery({
  key: ['mcp'],
  query: () => request({
    url: '/mcp/'
  })
})

const mcpFormatData = computed(() => {
  return mcpData.value?.data?.items??[]
})

</script>