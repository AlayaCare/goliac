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

    <el-row v-if="repositoryVariables.length > 0">
        &nbsp;
    </el-row>

    <el-row v-if="repositoryVariables.length > 0">
        <el-col :span="20" :offset="2">
            <el-card>
                <el-text>Variables</el-text>
                <el-table
                    :data="repositoryVariables"
                    :stripe="true"
                    :highlight-current-row="false"
                    :default-sort="{ prop: 'name', order: 'descending' }"
                >
                    <el-table-column prop="name" align="left" label="Name" sortable />
                    <el-table-column prop="value" align="left" label="Value" sortable />
                </el-table>
            </el-card>
        </el-col>
    </el-row>

    <el-row v-if="repositorySecrets.length > 0">
        &nbsp;
    </el-row>

    <el-row v-if="repositorySecrets.length > 0">
        <el-col :span="20" :offset="2">
            <el-card>
                <el-text>Secrets (to update, you will need the <a href="https://cli.github.com/" target="_blank">gh cli</a> installed)</el-text>
                <el-table
                    :data="repositorySecrets"
                    :stripe="true"
                    :highlight-current-row="false"
                    :default-sort="{ prop: 'name', order: 'descending' }"
                >
                    <el-table-column prop="name" :width="200" align="left" label="Name" sortable />
                    <el-table-column prop="name" align="left" label="Value" sortable >
                        <template #default="scope">
                           Use <code>`gh secret set {{ scope.row.name }} --repo {{ repository.organization }}/{{ repositoryid }} --body "value"`</code> to set a new value
                        </template>
                    </el-table-column>
                </el-table>
            </el-card>
        </el-col>
    </el-row>

    <el-row v-if="environmentVariables.length > 0">
        &nbsp;
    </el-row>

    <el-row v-if="environmentVariables.length > 0">
        <el-col :span="20" :offset="2">
            <el-card>
                <el-text>Environment Variables</el-text>
                <el-table
                    :data="environmentVariables"
                    :stripe="true"
                    :highlight-current-row="false"
                    :default-sort="{ prop: 'name', order: 'descending' }"
                >
                    <el-table-column prop="name" align="left" label="Name" sortable />
                    <el-table-column prop="value" align="left" label="Value" sortable />
                    <el-table-column prop="environment" align="left" label="Environment" sortable />
                </el-table>
            </el-card>
        </el-col>
    </el-row>

    <el-row v-if="environmentSecrets.length > 0">
        &nbsp;
    </el-row>

    <el-row v-if="environmentSecrets.length > 0">
        <el-col :span="20" :offset="2">
            <el-card>
                <el-text>Environment Secrets (to update, you will need the <a href="https://cli.github.com/" target="_blank">gh cli</a> installed)</el-text>
                <el-table
                    :data="environmentSecrets"
                    :stripe="true"
                    :highlight-current-row="false"
                    :default-sort="{ prop: 'name', order: 'descending' }"
                >
                    <el-table-column prop="name" align="left" label="Name" sortable />
                    <el-table-column prop="environment" align="left" label="Environment" sortable />
                    <el-table-column align="left" label="Value" sortable >
                        <template #default="scope">
                           Use <code>`gh secret set {{ scope.row.name }} --env {{ scope.row.environment }} --repo {{ repository.organization }}/{{ repositoryid }} --body "value"`</code> to set a new value
                        </template>
                    </el-table-column>
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
          repositoryVariables: [],
          repositorySecrets: [],
          environmentVariables: [],
          environmentSecrets: [],
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
                  this.repositoryVariables=repository.variables
                  this.repositorySecrets=repository.secrets
                  for (let environment of repository.environments) {
                    for (let variable of environment.variables) {
                      this.environmentVariables.push({
                        name: variable.name,
                        value: variable.value,
                        environment: environment.name
                      })
                    }
                    for (let secret of environment.secrets) {
                      this.environmentSecrets.push({
                        name: secret.name,
                        environment: environment.name
                      })
                    }
                  }
              }, handleErr.bind(this));
          },
      }
    };
  </script>
  