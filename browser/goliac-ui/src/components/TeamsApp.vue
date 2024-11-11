<template>
    <el-breadcrumb separator="/">
      <el-breadcrumb-item :to="{ path: '/' }">Goliac</el-breadcrumb-item>
      <el-breadcrumb-item :to="{ path: '/teams' }">teams</el-breadcrumb-item>
    </el-breadcrumb>
    <el-divider />
  
    <el-row>
      <el-col :span="20" :offset="2">
        <el-row>
          <el-table
              :data="teams"
              :stripe="true"
              :highlight-current-row="false"
              v-on:row-click="goToTeam"
              :default-sort="{ prop: 'path', order: 'descending' }"
          >
              <el-table-column prop="path" align="left" label="Team name" sortable />
              <el-table-column prop="owners" align="left" label="Nb owners" >
                <template #default="scope">
                    {{ scope.row.owners == null ? 0 : scope.row.owners.length }}
                </template>
              </el-table-column>
              <el-table-column prop="members" align="left" label="Nb members">
                <template #default="scope">
                    {{ scope.row.members == null ? 0 : scope.row.members.length }}
                </template>
              </el-table-column>
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
      name: "TeamsApp",
      components: {
      },
      data() {
        return {
          teams: [],
        };
      },
      created() {
        this.getTeams()
      },
      methods: {
        goToTeam(row) {
            this.$router.push({ name: "team", params: { teamId: row.name } });
        },
          getTeams() {
              Axios.get(`${API_URL}/teams`).then(response => {
                  let teams = response.data;
                  this.teams = teams
              }, handleErr.bind(this));
          },
      }
    };
  </script>
  