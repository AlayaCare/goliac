<template>
  <el-breadcrumb separator="/">
    <el-breadcrumb-item :to="{ path: '/' }">Goliac</el-breadcrumb-item>
    <el-breadcrumb-item :to="{ path: '/' }">dashboard</el-breadcrumb-item>
  </el-breadcrumb>
  <el-divider />

  <el-row>
    <el-col :span="20" :offset="2">
      <el-row>
        <el-table
            :data="statusTable"
            :stripe="true"
            :highlight-current-row="false"
            :default-sort="{ prop: 'title', order: 'descending' }"
        >
            <el-table-column width="150" prop="key" align="left" label="Key" sortable />
            <el-table-column prop="value" align="left" label="Value" />

        </el-table>
      </el-row>
      <el-row>
        <el-divider />
      </el-row>
      <el-row>
        <div class="flex-container">
            <el-text style="color:red;">Force a Github sync ? </el-text>
            <span> &nbsp; </span>
            <el-button @click="resync">Re-sync</el-button>
        </div>
      </el-row>
      <el-row>
        <span> &nbsp; </span>
      </el-row>
      <el-row>
        <div class="flex-container">
            <el-text style="color:red;">Invalidate the cache of remote Github objects? </el-text>
            <span> &nbsp; </span>
            <el-button @click="flushcache">Flush cache</el-button>
        </div>
      </el-row>  
      <el-row>
        <el-divider />
      </el-row>
      <el-row>
        <div class="flex-container">
            <el-text>Build version: {{ version }}</el-text>
        </div>
      </el-row>
    </el-col>
  </el-row>
</template>
  
<script>
  import Axios from "axios";
  
  import constants from "@/constants";
  import helpers from "@/helpers/helpers";
  import { h } from 'vue'
  import { ElNotification } from 'element-plus'
  
  const { handleErr } = helpers;
  
  const { API_URL } = constants;
  
  export default {
    name: "DashboardApp",
    components: {
    },
    data() {
      return {
        flushcacheVisible: false,
        statusTable: [],
        version: "",
      };
    },
    created() {
      this.getStatus()
    },
    methods: {
        getStatus() {
            Axios.get(`${API_URL}/status`).then(response => {
                let status = response.data;
                this.version = status.version;
                this.statusTable = [
                    {
                        key: "Last Sync",
                        value: status.lastSyncTime+" (UTC)",
                    },
                    {
                        key: "Last Sync Error",
                        value: status.lastSyncError,
                    },
                    {
                        key: "Nb Users",
                        value: status.nbUsers,
                    },
                    {
                        key: "Nb External Users",
                        value: status.nbUsersExternal,
                    },
                    {
                        key: "Nb Teams",
                        value: status.nbTeams,
                    },
                    {
                        key: "Nb Repositories",
                        value: status.nbRepos
                    },
                ]
        }, handleErr.bind(this));

        },
        flushcache() {
          Axios.post(`${API_URL}/flushcache`,{}).then(() => {
            ElNotification({
                title: 'Cache flushed',
                message: h('i', { style: 'color: teal' }, 'Github cache objects flushed'),
            })          
          }, handleErr.bind(this));
        },
        resync() {
          Axios.post(`${API_URL}/resync`,{}).then(() => {
            ElNotification({
                title: 'Sync started',
                message: h('i', { style: 'color: teal' }, 'Refresh this page in a moment'),
            })          
          }, handleErr.bind(this));
        },
    }
  };
</script>
