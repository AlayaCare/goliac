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
import WorkflowsApp from "@/components/WorkflowsApp.vue";
import WorkflowApp from "@/components/WorkflowApp.vue";

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
    path: "/workflows",
    name: "workflows",
    component: WorkflowsApp,
  },
  {
    path: "/workflows/:workflowName",
    name: "workflow",
    component: WorkflowApp,
  },
];

const router = createRouter({
  history: createWebHashHistory(),
  routes,
});

export default router;
