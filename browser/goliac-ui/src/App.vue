<template>
  <div id="app">
    <el-row style="background-color:#000000; height:40px;">
      <el-col :span="22" :offset="2" style="margin-top:8px;">
        <router-link :to="{ name: 'dashboard' }">
          <span style="color:#ffffff;">Goliac</span>
        </router-link>
      </el-col>
    </el-row>
    <el-row>
      &nbsp;
    </el-row>
    <div class="router-view-container">
      <router-view></router-view>
    </div>
  </div>
</template>

<script>

export default {
  name: 'App',
}
</script>

<style>
#app {
  font-family: Avenir, Helvetica, Arial, sans-serif;
  -webkit-font-smoothing: antialiased;
  -moz-osx-font-smoothing: grayscale;
  text-align: left;
  color: #2c3e50;
}

a:link { text-decoration: none; }
a:visited { text-decoration: none; }
a:hover { text-decoration: none; }
a:active { text-decoration: none; }


</style>