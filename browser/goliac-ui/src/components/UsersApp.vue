<template>
    <el-breadcrumb separator="/">
      <el-breadcrumb-item :to="{ path: '/' }">Goliac</el-breadcrumb-item>
      <el-breadcrumb-item :to="{ path: '/' }">users</el-breadcrumb-item>
    </el-breadcrumb>
    <el-divider />
  
    <el-row>
      <el-col :span="20" :offset="2">
        <el-row>
          <el-table
              :data="users"
              :stripe="true"
              :highlight-current-row="false"
              v-on:row-click="goToUser"
              :default-sort="{ prop: 'name', order: 'descending' }"
          >
              <el-table-column prop="name" align="left" label="Username" sortable />
              <el-table-column prop="githubid" align="left" label="Github Id" />
              <el-table-column prop="external" align="left" label="External" />
  
          </el-table>
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
      name: "UsersApp",
      components: {
      },
      data() {
        return {
          users: [],
        };
      },
      created() {
        this.getUsers()
      },
      methods: {
        goToUser(row) {
            this.$router.push({ name: "user", params: { userId: row.name } });
        },
          getUsers() {
              Axios.get(`${API_URL}/users`).then(response => {
                  let users = response.data.users;
                  this.users = users
              }, handleErr.bind(this));
          },
      }
    };
  </script>
  