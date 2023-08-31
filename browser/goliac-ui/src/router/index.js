import { createWebHashHistory, createRouter } from "vue-router";
import DashboardApp from "@/components/DashboardApp.vue";
import UsersApp from "@/components/UsersApp.vue";
import UserApp from "@/components/UserApp.vue";
import TeamsApp from "@/components/TeamsApp.vue";
import TeamApp from "@/components/TeamApp.vue";

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
    path: "/teams",
    name: "teams",
    component: TeamsApp,
  },
  {
    path: "/teams/:teamId",
    name: "team",
    component: TeamApp,
  },
];

const router = createRouter({
  history: createWebHashHistory(),
  routes,
});

export default router;
