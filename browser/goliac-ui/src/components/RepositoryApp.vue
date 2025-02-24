<template>
    <el-breadcrumb separator="/">
      <el-breadcrumb-item :to="{ path: '/' }">Goliac</el-breadcrumb-item>
      <el-breadcrumb-item :to="{ path: '/repositories' }">repositories</el-breadcrumb-item>
      <el-breadcrumb-item>{{ repositoryid }} repository</el-breadcrumb-item>
    </el-breadcrumb>
    <el-divider />
    
    <el-row>
        <el-col :span="20" :offset="2">
            <el-card>
                <template #header>
                    <div class="card-header">
                        <el-text>{{repositoryid}} repository</el-text>
                    </div>
                </template>
                <div class="flex-container">
                    <el-text>Visibility : </el-text>
                    <el-text>{{ repository.visibility}}</el-text>
                </div>
                <div class="flex-container">
                    <el-text>Archived : </el-text>
                    <el-text>{{ repository.archived}}</el-text>
                </div>
                <div class="flex-container">
                    <el-text>Auto Merge Allowed : </el-text>
                    <el-text>{{ repository.autoMergeAllowed}}</el-text>
                </div>
                <div class="flex-container">
                    <el-text>Delete Branch on Merge : </el-text>
                    <el-text>{{ repository.deleteBranchOnMerge}}</el-text>
                </div>
                <div class="flex-container">
                    <el-text>Allow Update Branch : </el-text>
                    <el-text>{{ repository.allowUpdateBranch}}</el-text>
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
            <el-text>Teams</el-text>

            <el-table
                :data="teams"
                :stripe="true"
                :highlight-current-row="false"
                v-on:row-click="goToTeam"
                :default-sort="{ prop: 'name', order: 'descending' }"
            >
                <el-table-column prop="name" align="left" label="Team Name" sortable />
                <el-table-column prop="access" align="left" label="Access" sortable />

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
            <el-text>Collaborators</el-text>

            <el-table
                :data="collaborators"
                :stripe="true"
                :highlight-current-row="false"
                v-on:row-click="goToCollaborator"
                :default-sort="{ prop: 'name', order: 'descending' }"
            >
                <el-table-column prop="name" align="left" label="Collaborator Name" sortable />
                <el-table-column prop="access" align="left" label="Access" sortable />

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
      name: "RepositoryApp",
      components: {
      },
      computed: {
        repositoryid() {
          return this.$route.params.repositoryId;
        },
      },

      data() {
        return {
          repository: {},
          teams: [],
          collaborators: [],
        };
      },
      created() {
        this.getRepository()
      },
      methods: {
        goToTeam(row) {
            this.$router.push({ name: "team", params: { teamId: row.name } });
        },
        goToCollaborator(row) {
            this.$router.push({ name: "collaborator", params: { collaboratorId: row.name } });
        },
          getRepository() {
              Axios.get(`${API_URL}/repositories/${this.repositoryid}`).then(response => {
                  let repository = response.data;
                  this.repository = repository
                  this.teams=repository.teams
                  this.collaborators=repository.collaborators
              }, handleErr.bind(this));
          },
      }
    };
  </script>
  