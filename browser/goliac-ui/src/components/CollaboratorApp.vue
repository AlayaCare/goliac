<template>
    <el-breadcrumb separator="/">
      <el-breadcrumb-item :to="{ path: '/' }">Goliac</el-breadcrumb-item>
      <el-breadcrumb-item :to="{ path: '/collaborators' }">external collaborators</el-breadcrumb-item>
      <el-breadcrumb-item>{{ collaboratorid }}</el-breadcrumb-item>
    </el-breadcrumb>
    <el-divider />
    
    <el-row>
        <el-col :span="20" :offset="2">
            <el-card>
                <template #header>
                    <div class="card-header">
                        <el-text>{{collaboratorid}}</el-text>
                    </div>
                </template>
                <div class="flex-container">
                    <el-text>Github id : </el-text>
                    <el-text>{{ collaborator.githubid}}</el-text>
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
      name: "CollaboratorApp",
      components: {
      },
      computed: {
        collaboratorid() {
          return this.$route.params.collaboratorId;
        },
      },

      data() {
        return {
          collaborator: {},
          repositories: [],
        };
      },
      created() {
        this.getCollaborator()
      },
      methods: {
        goToRepository(row) {
            this.$router.push({ name: "repository", params: { repositoryId: row.name } });
        },
          getCollaborator() {
              Axios.get(`${API_URL}/collaborators/${this.collaboratorid}`).then(response => {
                  let collaborator = response.data;
                  this.collaborator = collaborator
                  this.repositories=collaborator.repositories
              }, handleErr.bind(this));
          },
      }
    };
  </script>
  