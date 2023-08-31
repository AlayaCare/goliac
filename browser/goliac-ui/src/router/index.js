import { createWebHashHistory, createRouter } from "vue-router";
import DashboardApp from "@/components/DashboardApp.vue";
import UsersApp from "@/components/UsersApp.vue";
import UserApp from "@/components/UserApp.vue";

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
];

const router = createRouter({
  history: createWebHashHistory(),
  routes,
});

export default router;
