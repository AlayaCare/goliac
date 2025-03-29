import { createWebHashHistory, createRouter } from "vue-router";
import DashboardApp from "@/components/DashboardApp.vue";
import UsersApp from "@/components/UsersApp.vue";
import UserApp from "@/components/UserApp.vue";
import CollaboratorsApp from "@/components/CollaboratorsApp.vue";
import CollaboratorApp from "@/components/CollaboratorApp.vue";
import TeamsApp from "@/components/TeamsApp.vue";
import TeamApp from "@/components/TeamApp.vue";
import RepositoriesApp from "@/components/RepositoriesApp.vue";
import RepositoryApp from "@/components/RepositoryApp.vue";
import ForcemergeWorkflowsApp from "@/components/ForcemergeWorkflowsApp.vue";
import ForcemergeWorkflowApp from "@/components/ForcemergeWorkflowApp.vue";

const routes = [
  {
    path: "/",
    name: "dashboard",
    component: DashboardApp,
  },
  {
    path: "/users",
    name: "users",
    component: UsersApp,
  },
  {
    path: "/users/:userId",
    name: "user",
    component: UserApp,
  },
  {
    path: "/collaborators",
    name: "collaborators",
    component: CollaboratorsApp,
  },
  {
    path: "/collaborators/:collaboratorId",
    name: "collaborator",
    component: CollaboratorApp,
  },
  {
    path: "/teams",
    name: "teams",
    component: TeamsApp,
  },
  {
    path: "/teams/:teamId",
    name: "team",
    component: TeamApp,
  },
  {
    path: "/repositories",
    name: "repositories",
    component: RepositoriesApp,
  },
  {
    path: "/repositories/:repositoryId",
    name: "repository",
    component: RepositoryApp,
  },
  {
    path: "/forcemergeworkflows",
    name: "workflows",
    component: ForcemergeWorkflowsApp,
  },
  {
    path: "/forcemergeworkflows/:workflowName",
    name: "workflow",
    component: ForcemergeWorkflowApp,
  },
];

const router = createRouter({
  history: createWebHashHistory(),
  routes,
});

export default router;
