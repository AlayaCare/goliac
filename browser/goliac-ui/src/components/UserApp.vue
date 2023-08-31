<template>
    <el-breadcrumb separator="/">
      <el-breadcrumb-item :to="{ path: '/' }">Goliac</el-breadcrumb-item>
      <el-breadcrumb-item :to="{ path: '/users' }">users</el-breadcrumb-item>
      <el-breadcrumb-item>{{ userid }}</el-breadcrumb-item>
    </el-breadcrumb>
    <el-divider />
    
    <el-row>
        <el-col :span="20" :offset="2">
            <el-card>
                <template #header>
                    <div class="card-header">
                        <el-text>{{userid}}</el-text>
                    </div>
                </template>
                <div class="flex-container">
                    <el-text>Github id : </el-text>
                    <el-text>{{ user.githubid}}</el-text>
                </div>
                <div class="flex-container">
                    <el-text>External : </el-text>
                    <el-text> {{ user.external}}</el-text>
                </div>
            </el-card>
        </el-col>
    </el-row>  

    <el-row>
        &nbsp;
    </el-row>

    <el-row>
      <el-col :span="20" :offset="2">
        <el-card>
          <el-table
              :data="teams"
              :stripe="true"
              :highlight-current-row="false"
              v-on:row-click="goToTeam"
              :default-sort="{ prop: 'name', order: 'descending' }"
          >
              <el-table-column prop="name" align="left" label="Team" sortable />
  
          </el-table>
        </el-card>
      </el-col>
    </el-row>
    <el-row>
        &nbsp;
    </el-row>

    <el-row>
      <el-col :span="20" :offset="2">
        <el-card>
          <el-table
              :data="repositories"
              :stripe="true"
              :highlight-current-row="false"
              v-on:row-click="goToRepository"
              :default-sort="{ prop: 'name', order: 'descending' }"
          >
              <el-table-column prop="name" align="left" label="Repository" sortable />
              <el-table-column prop="public" align="left" label="Public" />
              <el-table-column prop="archived" align="left" label="Archived" />
  
          </el-table>
        </el-card>
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
      name: "UserApp",
      components: {
      },
      computed: {
        userid() {
          return this.$route.params.userId;
        },
      },

      data() {
        return {
          user: {},
          repositories: [],
          teams: [],
        };
      },
      created() {
        this.getUser()
      },
      methods: {
        goToTeam(row) {
            this.$router.push({ name: "team", params: { teamId: row.name } });
        },
        goToRepository(row) {
            this.$router.push({ name: "repository", params: { repositoryId: row.name } });
        },
          getUser() {
              Axios.get(`${API_URL}/users/${this.userid}`).then(response => {
                  let user = response.data;
                  this.user = user
                  this.repositories=user.repositories
                  this.teams=user.teams
              }, handleErr.bind(this));
          },
      }
    };
  </script>
  