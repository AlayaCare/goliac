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

  <el-breadcrumb separator="/">
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

        <span style="color:red">Invalidate the cache of remote Github objects? </span> <el-button @click="flushcache">Flush cache</el-button>
      </el-row>
    </el-col>
  </el-row>
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
        statusTable: [],
      };
    },
    created() {
      this.getStatus()
    },
    methods: {
        getStatus() {
            Axios.get(`${API_URL}/status`).then(response => {
                let status = response.data;
                this.statusTable = [
                    {
                        key: "Last Sync",
                        value: status.lastSyncTime,
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
            this.flushcacheVisible=false
          }, handleErr.bind(this));
        }
    }
  };
</script>
