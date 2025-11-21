# Deployment on Railway or Render

- [Railway](#railway)
- [Render](#render)

## Railway
- In [Railway's dashboard](https://railway.com/dashboard), click on the purple `New` button
- Choose `Github Repository`
<div align="center">
  <img width="1048" height="1137" alt="image" src="https://github.com/user-attachments/assets/cf8c8bdb-e3c6-401c-a1a7-4b5d2d0788e1" />
</div>

- Choose your repo

<div align="center">
  <img width="950" height="750" alt="image" src="https://github.com/user-attachments/assets/a4c95bed-c06d-479e-b8cb-5a4599b07c44" />
  <i>Mine above is Aegirus</i>
</div>

- Click on settings, all should be default instead of domain
- Click on `Generate Domain` and set the port
<img width="1412" height="524" alt="image" src="https://github.com/user-attachments/assets/340cf0e8-7c44-4483-9293-4659ca523944" />
<img width="976" height="521" alt="image" src="https://github.com/user-attachments/assets/efe52ed6-2fa8-4b97-ab82-9a2103e46ac5" />

- You will get a domain like below
<img width="953" height="430" alt="image" src="https://github.com/user-attachments/assets/341b4451-deba-46f7-acc1-a14d2a8c3bf2" />

- **This is the most important part, click on variables**
- Click on `Raw editor` and paste your `.env` into it
<img width="1348" height="750" alt="image" src="https://github.com/user-attachments/assets/ad05f1fc-54f9-4689-a203-85964fd01350" />

> [!IMPORTANT]
> Click on the `Deploy` button to apply all variables from the settting  
> If you don't click on this button but deploy by default only, your environment variables won't be used
> REMEMBER TO SET THE TRUSTED_PROXIES TO `true`  

<div align="center">
  <img width="643" height="206" alt="image" src="https://github.com/user-attachments/assets/f5f42e08-2ec5-4045-bb53-ece05e21f919" />
</div>

## Render

- In [Render's dashboard](https://dashboard.render.com/), click on the `Add new` button
- Click on `Web service`
<div align="center">
  <img width="427" height="683" alt="image" src="https://github.com/user-attachments/assets/68ece456-65e1-4744-96f4-70e86f5b3a2a" />
</div>
  
- Choose your repo

<div align="center">
  <img width="1708" height="264" alt="image" src="https://github.com/user-attachments/assets/c65872d4-da5a-430c-9614-d02058913e16" />
  <i>Mine above is Aegirus</i>
</div>

- Set any name you like
- Choose branch you want (should be main)
- Keep using `Docker`
<img width="1676" height="830" alt="image" src="https://github.com/user-attachments/assets/0f4862dc-d9d3-4a20-96cd-f07f0b97e92a" />

- Choose **Free** plan
<img width="2448" height="867" alt="image" src="https://github.com/user-attachments/assets/64da0c13-5506-41ed-9a40-cda985489623" />

- Scroll down and click on `Add from .env`
<div align="center">
  <img width="2444" height="412" alt="image" src="https://github.com/user-attachments/assets/7a1b25ca-9b97-43a7-9fa2-ed555a7760de" />
</div>
- Paste all your variables
<div align="center">
  <img width="865" height="646" alt="image" src="https://github.com/user-attachments/assets/28623381-5ac5-4ee6-b56a-0764e34cd34c" />
</div>

- You will get something like this
<img width="2331" height="1251" alt="image" src="https://github.com/user-attachments/assets/2bfa660d-4269-4cba-a89f-c72e5d2db14e" />

> [!IMPORTANT]
> REMEMBER TO SET THE TRUSTED_PROXIES TO `true`  
