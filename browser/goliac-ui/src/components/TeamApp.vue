<template>
    <el-breadcrumb separator="/">
      <el-breadcrumb-item :to="{ path: '/' }">Goliac</el-breadcrumb-item>
      <el-breadcrumb-item :to="{ path: '/teams' }">teams</el-breadcrumb-item>
      <el-breadcrumb-item>{{ teamid }} team</el-breadcrumb-item>
    </el-breadcrumb>
    <el-divider />
    
    <el-row>
        <el-col :span="20" :offset="2">
            <el-card>
                <template #header>
                    <div class="card-header">
                        <el-text>{{teamid}} team</el-text>
                    </div>
                </template>
                <el-text>Team Owners</el-text>

                <el-table
                    :data="owners"
                    :stripe="true"
                    :highlight-current-row="false"
                    v-on:row-click="goToUser"
                    :default-sort="{ prop: 'name', order: 'descending' }"
                >
                    <el-table-column prop="name" align="left" label="Owner Name" sortable />
                    <el-table-column prop="githubid" align="left" label="Github Id" />
                    <el-table-column prop="external" align="left" label="External" />
        
                </el-table>
                <br/>
                <br/>
                <el-text>Team Members</el-text>
                <el-table
                    :data="members"
                    :stripe="true"
                    :highlight-current-row="false"
                    v-on:row-click="goToUser"
                    :default-sort="{ prop: 'name', order: 'descending' }"
                >
                <el-table-column prop="name" align="left" label="Member Name" sortable />
                    <el-table-column prop="githubid" align="left" label="Github Id" />
                    <el-table-column prop="external" align="left" label="External" />
        
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
      name: "TeamApp",
      components: {
      },
      computed: {
        teamid() {
          return this.$route.params.teamId;
        },
      },

      data() {
        return {
          team: {},
          repositories: [],
          owners: [],
          members: [],
        };
      },
      created() {
        this.getTeam()
      },
      methods: {
        goToUser(row) {
            this.$router.push({ name: "user", params: { userId: row.name } });
        },
        goToRepository(row) {
            this.$router.push({ name: "repository", params: { repositoryId: row.name } });
        },
          getTeam() {
              Axios.get(`${API_URL}/teams/${this.teamid}`).then(response => {
                  let team = response.data;
                  this.team = team
                  this.repositories=team.repositories
                  this.owners = team.owners
                  this.members = team.members
              }, handleErr.bind(this));
          },
      }
    };
  </script>
  