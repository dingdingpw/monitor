import { createApp } from 'vue'
import ArcoVue from '@arco-design/web-vue';
import App from './App.vue';
import '@arco-design/web-vue/dist/arco.css';
import 'flag-icons-svg/css/flag-icons.css';

import ArcoVueIcon from '@arco-design/web-vue/es/icon';
import i18n from "@/locales";

const app = createApp(App);
app.use(i18n)
app.use(ArcoVue);
app.use(ArcoVueIcon);
app.mount('#app');