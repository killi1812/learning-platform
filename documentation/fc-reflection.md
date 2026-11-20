# AI Reflection — Fran
<!--toc:start-->

[AI reflection](#ai-reflection)
[Where and why](#where-and-why)
[Used](#used)
[Not Used](#not-used)
[Tools](#tools)
[Quality assurance](#quality-assurance)
[Lessons learned](#lessons-learned) <!--toc:end-->
Your documentation must contain a short reflection on your use of AI. If you did not use AI, state this and briefly explain why. The reflection should address the following points:

Where and why: For which tasks and parts of the project did you use AI, and why did you decide to use it there (or not)?
Tools: Which tools and models did you use (e.g. chat assistants, code completion in the IDE, or autonomous coding agents), and how did you work with them?
Quality assurance: Which measures did you take to evaluate the quality of the AI output (e.g. code reviews, tests, static analysis, or checking against official documentation)?
Lessons learned: Where did AI help you, where did it fail or mislead you, and what would you do differently next time?
Where and why
Used
Implementation

After discussing a feature with colleagues, I would prompt AI to plan the implementation while adhering to the project's existing patterns and interfaces. Whenever a new pattern was needed, I explicitly specified it.
CI/CD

The .gitlab-ci.yml pipeline was implemented by AI.
UI

Most of the implementation was done with AI, with only the main project setup handled manually (e.g., Axios APIs in client/src/api/<model_name>.ts were implemented by AI).
Technology research

Used for initial research to explore which technologies fit a problem and understand their strengths and weaknesses. When deeper knowledge was needed, I read the documentation.
The main benefits were faster implementation and bridging knowledge gaps when working with new technologies.

Not Used
Project setup

While the project setup had some AI assistance (such as renaming and minor cleanups), it is based on a template I reuse across multiple projects. I copied and adapted it to fit this project's specifics while keeping the core patterns intact.
I wanted a setup I already understood and could easily explain to my colleagues, rather than prompting AI to generate a Go and Vue web app from scratch.
Code design choices

The core design patterns in the codebase—such as dependency injection, workers, and API interfaces—were designed manually for the same reason.
Feature design

Every feature was discussed with colleagues beforehand so that everyone stayed aligned on the application's functionality.
AI rules

Following the principle of "garbage in, garbage out," I wrote the AI rules myself—if I let AI write its own rules, I cannot expect them to be good or ensure it behaves the way I want.
Tools
I used Antigravity-cli almost exclusively, as I have a paid Google subscription. My workflow was to have it create an implementation plan first. After verifying the plan, I would let the agent implement it autonomously. The main restrictions were on the commands it could execute. I allowed it to run any task command so it could execute tests and verify its own results. I also started a new chat for each feature and had it perform a code review (in a separate new chat) once the feature was complete. Additionally, I used the OpenWolf tool for better memory management, though I doubt it had a meaningful impact. I primarily used Gemini Flash 3.8, which was the best model available until Argon was released; after that, to conserve my Argon quota, I planned features with Argon and implemented them with Flash.

Quality assurance
Quality was ensured through both AI and human code reviews, planning before implementation, a solid project structure, and handwritten rules.

Lessons learned
It is hard to keep up with AI-generated changes, leaving me unsure how maintainable this codebase really is. Because AI writes code so quickly, it is not always reliable at following the project structure, or at least it needs explicit guidance on what to do.