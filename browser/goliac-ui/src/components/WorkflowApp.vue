<template>
    <el-breadcrumb separator-class="el-icon-arrow-right">
        <el-breadcrumb-item :to="{ path: '/' }">Goliac</el-breadcrumb-item>
        <el-breadcrumb-item :to="{ path: '/workflows' }">Workflows</el-breadcrumb-item>
        <el-breadcrumb-item :to="{ path: '/{{ workflowName }}' }">{{ workflowName }}</el-breadcrumb-item>
    </el-breadcrumb>
    <el-divider />
    <el-row>
        <el-col :span="20" :offset="2">
            <div class="wizard-container">
                <el-steps :active="activeStep" finish-status="success">
                    <el-step title="Collect informations"></el-step>
                    <el-step title="Submit"></el-step>
                    <el-step title="Get back status"></el-step>
                </el-steps>

                <div class="step-content" style="margin: 20px 0;">
                    <div v-if="activeStep === 0">
                        <div v-if="workflow_type === 'forcemerge'">
                            <el-form :model="form" label-width="auto" style="max-width: 600px">
                                <el-form-item label="PR URL">
                                    <el-input v-model="pr_url" placeholder="PR URL" />
                                </el-form-item>
                                <el-form-item label="Associated justification">
                                    <el-input
                                        v-model="explanation"
                                        type="textarea"
                                        rows="4"
                                        placeholder="Associated justification"
                                    />
                                </el-form-item>
                            </el-form>
                            <div class="wizard-footer">
                                <el-button :disabled="explanation.length==0 || pr_url.length<10" type="success" @click="submitForcemerge">Submit</el-button>
                            </div>
                        </div>
                        <div v-if="workflow_type === 'noop'">
                            <el-form :model="form" label-width="auto" style="max-width: 600px">
                                <el-form-item label="Associated justification">
                                    <el-input
                                        v-model="explanation"
                                        type="textarea"
                                        rows="4"
                                        placeholder="Associated justification"
                                    />
                                </el-form-item>
                            </el-form>
                            <div class="wizard-footer">
                                <el-button :disabled="explanation.length==0" type="success" @click="submitTest">Submit</el-button>
                            </div>
                        </div>
                    </div>

                    <div v-if="activeStep === 1">
                        <el-skeleton :rows="5" animated />
                    </div>
                    <div v-if="activeStep === 2">
                        <div v-if="result_error">
                            <el-alert
                                title="Error"
                                type="error"
                                :closable="false"
                                :description="message"
                            />
                        </div>
                        <div v-else>
                            <p>{{ message }}</p>
                            <p v-for="(url,index) in tracking_urls"
                                :key="index"
                            >
                                <a :href="url" target="_blank">{{ url }}</a>
                            </p>
                        </div>
                    </div>
                </div>
            </div>
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
    name: "WorkflowApp",
    components: {
    },
    computed: {
      workflowName() {
        return this.$route.params.workflowName;
      },
    },
    data() {
      return {
        activeStep: 0,
        workflow_type: "",
        pr_url: "",
        explanation: "",
        message: "",
        tracking_urls: [],
        result_error: false,
      };
    },
    created() {
      this.getWorkflowType();
    },
    methods: {
      getWorkflowType() {
        // Get the workflow type from the API
        Axios.get(`${API_URL}/auth/workflows/${this.workflowName}`)
          .then(response => {
            this.workflow_type = response.data.workflow_type;
          }, handleErr.bind(this));
      },
      submitForcemerge() {
        this.activeStep=1;
        // Final action after wizard is done
        Axios.post(`${API_URL}/auth/workflows/${this.workflowName}`,
            {
                explanation: this.explanation,
                properties: [
                    {
                        name: "pr_url",
                        value: this.pr_url,
                    },
                ],
            }
        ).then(response => {
          let result = response.data;

          this.activeStep=2;
          this.message = result.message;
          this.tracking_urls = result.tracking_urls;
        }, error => {
            this.activeStep=2;
            this.result_error = true;
            this.message = error.response.data.message;
        });
      },
      submitTest() {
        this.activeStep=1;
        // Final action after wizard is done
        Axios.post(`${API_URL}/auth/workflows/${this.workflowName}`,
            {
                explanation: this.explanation,
                properties: [],
            }
        ).then(response => {
          let result = response.data;

          this.activeStep=2;
          this.message = result.message;
          this.tracking_urls = result.tracking_urls;
        }, error => {
            this.activeStep=2;
            this.result_error = true;
            this.message = error.response.data.message;
        });
      }
    }
  };
</script>

<style scoped>
  .wizard-container {
    margin: 20px;
  }

  .step-content {
    margin: 20px 0;
  }

  .m4 {
  margin-bottom: 16px;
  }

  .wizard-footer {
    margin-top: 20px;
  }
</style>
