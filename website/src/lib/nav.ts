import { docs } from './docs';
import { docsPages } from './site';

// Pages built by a page of their own rather than from content/docs.
const ownPages: readonly string[] = ['introduction', 'examples'];

// The pages the navigation lists: those that exist, in the order of docsPages.
export const navPages = docsPages.filter((page) => ownPages.includes(page.id) || docs.some((doc) => doc.id === page.id));
