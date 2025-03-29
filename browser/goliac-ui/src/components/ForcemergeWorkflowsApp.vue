<script setup>
import { FontAwesomeIcon } from '@fortawesome/vue-fontawesome'
import { faFireExtinguisher } from '@fortawesome/free-solid-svg-icons';
</script>

<template>
  <el-breadcrumb separator="/">
    <el-breadcrumb-item :to="{ path: '/' }">Goliac</el-breadcrumb-item>
    <el-breadcrumb-item :to="{ path: '/' }">PR workflows</el-breadcrumb-item>
  </el-breadcrumb>
  <el-divider />
  <el-row>
    <el-col :span="20" :offset="2">
      <el-row>
        &nbsp;
      </el-row>
      <el-row :gutter="20">
        <el-col
          v-for="(item, index) in forcemergeworkflows"
          :key="index"
          :xs="24" :sm="12" :md="8" :lg="6" class="mb-4"
        >
          <el-card shadow="hover" class="clickable-card" @click="handleClick(item)">
            <template #header>
              <div class="card-header">{{ item.workflow_name }}</div>
            </template>
            <div class="card-content">
              <font-awesome-icon :icon="faFireExtinguisher" class="extinguisher-icon" />
              <p>{{ item.workflow_description }}</p>
            </div>
          </el-card>
        </el-col>
      </el-row>
    </el-col>
  </el-row>
</template>

<script>
  import Axios from "axios";
  
  import constants from "@/constants";
  import helpers from "@/helpers/helpers";
  // import { ElMessage } from 'element-plus';

  const { handleErr } = helpers;
  
  const { API_URL } = constants;
  
  export default {
    name: "ForcemergeWorkflowsApp",
    components: {
    },
    data() {
      return {
        forcemergeworkflows: [],
      };
    },
    mounted() {
      this.getForcemergeWorkflows()
    },
    methods: {
      getForcemergeWorkflows() {
        Axios.get(`${API_URL}/auth/workflows_forcemerge`).then(response => {
          this.forcemergeworkflows = response.data;
        }, handleErr.bind(this));
      },
      handleClick(item) {
        this.$router.push({ name: "workflow", params: { workflowName: item.workflow_name } });

        // ElMessage.success('Card clicked! '+item.workflow_name);
      },
    }
  };
</script>

<style scoped>
.clickable-card {
  cursor: pointer;
  transition: transform 0.2s ease-in-out;
  font-family: 'Helvetica Neue', Helvetica, Arial, sans-serif;

  height: 250px; /* Fixed height */
  display: flex;
  flex-direction: column;
  justify-content: space-between; /* Ensures content is spaced properly */

  background-color: #f5f7fa; /* Light Gray */
}

.clickable-card:hover {
  transform: scale(1.02);
}

.card-header {
  text-align: left;
}

::v-deep(.el-card__header) {
  background-color: #ffffff;
  padding: 12px;
  border-radius: 4px;
}

::v-deep(.el-card__body) {
  flex-grow: 1;
  display: flex;
  flex-direction: column;
  justify-content: center; /* Centers content vertically */
  text-align: center;
}

/* Fire extinguisher icon styling */
.extinguisher-icon {
  font-size: 30px;
  color: red;
  margin-bottom: 8px;
}

.mb-4 {
  margin-bottom: 16px;
}
</style>
