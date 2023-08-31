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
                    <el-text>Public : </el-text>
                    <el-text>{{ repository.public}}</el-text>
                </div>
                <div class="flex-container">
                    <el-text>Archived : </el-text>
                    <el-text>{{ repository.archived}}</el-text>
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
            <el-text>Team with write access</el-text>
            <el-table
                :data="writers"
                :stripe="true"
                :highlight-current-row="false"
                v-on:row-click="goToTeam"
                :default-sort="{ prop: 'name', order: 'descending' }"
            >
            <el-table-column prop="name" align="left" label="Team Name" sortable />

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
            <el-text>Team with read access</el-text>

            <el-table
                :data="readers"
                :stripe="true"
                :highlight-current-row="false"
                v-on:row-click="goToTeam"
                :default-sort="{ prop: 'name', order: 'descending' }"
            >
                <el-table-column prop="name" align="left" label="Team Name" sortable />

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
          readers: [],
          writers: [],
        };
      },
      created() {
        this.getRepository()
      },
      methods: {
        goToTeam(row) {
            this.$router.push({ name: "team", params: { teamId: row.name } });
        },
          getRepository() {
              Axios.get(`${API_URL}/repositories/${this.repositoryid}`).then(response => {
                  let repository = response.data;
                  this.repository = repository
                  this.readers=repository.readers
                  this.writers=repository.writers
              }, handleErr.bind(this));
          },
      }
    };
  </script>
  