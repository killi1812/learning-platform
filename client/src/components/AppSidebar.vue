<template>
  <v-navigation-drawer permanent width="220">
    <v-list density="compact">
      <v-list-item title="Dashboard" prepend-icon="mdi-home" />
      <v-list-item title="My Courses" prepend-icon="mdi-book-open" />
      <v-list-item title="Messages" prepend-icon="mdi-message" />
      <v-list-item title="Groups" prepend-icon="mdi-account-group" />
      <v-list-item title="Grades" prepend-icon="mdi-chart-bar" />
    </v-list>
    <v-divider />
    <v-list density="compact">
      <v-list-item title="Settings" prepend-icon="mdi-cog" />
      <v-list-item title="Logout" prepend-icon="mdi-logout" />
    </v-list>
  </v-navigation-drawer>
</template>

<script lang="ts" setup></script>
