// Entry point: mount the root App component into the page.
// Svelte 5 uses mount() instead of Svelte 4's `new App({ target })`.
import { mount } from 'svelte';
import './app.css';
import App from './App.svelte';

const app = mount(App, {
  target: document.getElementById('app'),
});

export default app;
