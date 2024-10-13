<template>
    <el-breadcrumb separator="/">
      <el-breadcrumb-item :to="{ path: '/' }">Goliac</el-breadcrumb-item>
      <el-breadcrumb-item :to="{ path: '/repositories' }">repositories</el-breadcrumb-item>
    </el-breadcrumb>
    <el-divider />
  
    <el-row>
      <el-col :span="20" :offset="2">
        <el-row>
          <el-table
              :data="repositories"
              :stripe="true"
              :highlight-current-row="false"
              v-on:row-click="goToRepository"
              :default-sort="{ prop: 'name', order: 'descending' }"
          >
              <el-table-column prop="name" align="left" label="Repository name" sortable />
              <el-table-column prop="public" align="left" label="Public" sortable />
              <el-table-column prop="archived" align="left" label="Archived" sortable />
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
      name: "RepositoriesApp",
      components: {
      },
      data() {
        return {
          repositories: [],
        };
      },
      created() {
        this.getRepositories()
      },
      methods: {
        goToRepository(row) {
            this.$router.push({ name: "repository", params: { repositoryId: row.name } });
        },
          getRepositories() {
              Axios.get(`${API_URL}/repositories`).then(response => {
                  let repositories = response.data;
                  this.repositories = repositories
              }, handleErr.bind(this));
          },
      }
    };
  </script>
  