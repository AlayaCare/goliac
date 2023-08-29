<script setup>
import { User, MessageBox, Folder } from '@element-plus/icons-vue'
</script>

<template>
 <div id="app">
    <el-container>
      <el-aside width="130px">
        <el-menu>
          <el-menu-item index="1" style="justify-content: center">
            <template #title>
              <img src="/logo.png">
            </template>
          </el-menu-item>
          <el-menu-item index="2">
            <template #title>
              <el-icon :size="16"><User /></el-icon>Users
            </template>
          </el-menu-item>
          <el-menu-item index="3">
            <template #title>
              <el-icon :size="16"><MessageBox /></el-icon>Teams
            </template>
          </el-menu-item>
          <el-menu-item index="4">
            <template #title>
              <el-icon :size="16"><Folder /></el-icon>Repositories
            </template>
          </el-menu-item>
        </el-menu>
      </el-aside>

      <el-container>
        <el-main>
          <router-view></router-view>
        </el-main>
      </el-container>
    </el-container>
  </div>
</template>

<script>
export default {
  name: 'App',
}

</script>

<style scoped>
.layout-container .el-header {
  position: relative;
  background-color: var(--el-color-primary-light-7);
  color: var(--el-text-color-primary);
}
.layout-container .el-aside {
  color: var(--el-text-color-primary);
  background: var(--el-color-primary-light-8);
}
</style>
