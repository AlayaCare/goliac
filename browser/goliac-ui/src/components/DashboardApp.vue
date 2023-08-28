<template>
      <el-dialog v-model="flushcacheVisible" title="Cache flushed" width="60%">
        <span class="dialog-content">
            The cache has been flushed, next sync will load from Github
            </span>
    <template #footer>
      <span class="dialog-footer">
        <el-button type="primary" @click="flushcacheVisible = false">Ok</el-button>
      </span>
    </template>
  </el-dialog>
  test
</template>
  
<script>
  import Axios from "axios";
  
  import constants from "@/constants";
  import helpers from "@/helpers/helpers";
  
  const { handleErr } = helpers;
  
  const { API_URL } = constants;
  
  export default {
    name: "DashboardApp",
    components: {
    },
    data() {
      return {
        flushcacheVisible: false,
      };
    },
    methods: {
        flushcache() {
        Axios.post(`${API_URL}/flushcache`,{}).then(() => {
          this.flushcacheVisible=false
        }, handleErr.bind(this));
      }
    }
  };
</script>
