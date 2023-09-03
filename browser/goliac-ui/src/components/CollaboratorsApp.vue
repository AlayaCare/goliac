<template>
    <el-breadcrumb separator="/">
      <el-breadcrumb-item :to="{ path: '/' }">Goliac</el-breadcrumb-item>
      <el-breadcrumb-item :to="{ path: '/collaborators' }">external collaborators</el-breadcrumb-item>
    </el-breadcrumb>
    <el-divider />
  
    <el-row>
      <el-col :span="20" :offset="2">
        <el-row>
          <el-table
              :data="collaborators"
              :stripe="true"
              :highlight-current-row="false"
              v-on:row-click="goToCollaborator"
              :default-sort="{ prop: 'name', order: 'descending' }"
          >
              <el-table-column prop="name" align="left" label="Username" sortable />
              <el-table-column prop="githubid" align="left" label="Github Id" />
  
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
      name: "CollaboratorsApp",
      components: {
      },
      data() {
        return {
          collaborators: [],
        };
      },
      created() {
        this.getCollaborators()
      },
      methods: {
        goToCollaborator(row) {
            this.$router.push({ name: "collaborator", params: { collaboratorId: row.name } });
        },
          getCollaborators() {
              Axios.get(`${API_URL}/collaborators`).then(response => {
                  let collaborators = response.data;
                  this.collaborators = collaborators
              }, handleErr.bind(this));
          },
      }
    };
  </script>
  