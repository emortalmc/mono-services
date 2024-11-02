Idea for checking if there are differences:
- Use git diff to check if a module has been modified
- Analyse the module dependencies to see what modules are affected by the changes
- Run the build process for all affected modules, taking into account the dependency graph
